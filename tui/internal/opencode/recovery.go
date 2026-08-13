package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"codea/tui/internal/runtime"
)

// SessionTracker maintains local state of observed sessions/messages/parts
// from the SSE event stream. After a disconnect and reconnect, it can query
// the OpenCode HTTP API to find missed items and generate compensation events.
type SessionTracker struct {
	mu       sync.Mutex
	sessions map[string]*trackedSession
	seq      int64
	// recoveredMsgIDs / recoveredPartIDs are the message/part IDs compensated
	// by the most recent Recover() pass. Live events that race into the
	// reconnect buffer during recovery and match these IDs are duplicates of
	// the compensation events and are suppressed once by ShouldSuppressLive.
	recoveredMsgIDs  map[string]bool
	recoveredPartIDs map[string]bool
}

type trackedSession struct {
	messages map[string]bool
	parts    map[string]bool
}

// NewSessionTracker creates a new tracker.
func NewSessionTracker() *SessionTracker {
	return &SessionTracker{
		sessions:         make(map[string]*trackedSession),
		recoveredMsgIDs:  make(map[string]bool),
		recoveredPartIDs: make(map[string]bool),
	}
}

// Record updates the tracker with state from an observed event.
func (t *SessionTracker) Record(ev runtime.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ev.Sequence > t.seq {
		t.seq = ev.Sequence
	}

	if ev.SessionID == "" {
		return
	}
	s := t.sessions[ev.SessionID]
	if s == nil {
		s = &trackedSession{
			messages: make(map[string]bool),
			parts:    make(map[string]bool),
		}
		t.sessions[ev.SessionID] = s
	}
	if ev.MessageID != "" {
		s.messages[ev.MessageID] = true
	}
	if ev.PartID != "" {
		s.parts[ev.PartID] = true
	}
}

// LastSeq returns the highest sequence number observed.
func (t *SessionTracker) LastSeq() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.seq
}

// ShouldSuppressLive reports whether a live event is a duplicate of a
// message/part that was just compensated by the most recent Recover() pass.
// The matching ID is consumed, so a subsequent genuine update for the same
// ID is not suppressed.
func (t *SessionTracker) ShouldSuppressLive(ev runtime.Event) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch ev.Type {
	case CodeaEventMessageUpdated:
		if ev.MessageID != "" && t.recoveredMsgIDs[ev.MessageID] {
			delete(t.recoveredMsgIDs, ev.MessageID)
			return true
		}
	case CodeaEventPartUpdated:
		if ev.PartID != "" && t.recoveredPartIDs[ev.PartID] {
			delete(t.recoveredPartIDs, ev.PartID)
			return true
		}
	}
	return false
}

// Recover queries the OpenCode API for current session/message state and
// returns compensation events for any sessions or messages that are new
// since the last observation.
func (t *SessionTracker) Recover(ctx context.Context, client *HTTPClient) []runtime.Event {
	t.mu.Lock()
	defer t.mu.Unlock()

	var events []runtime.Event

	// Reset the per-pass dedup sets; repopulated below for the messages/parts
	// this pass compensates.
	t.recoveredMsgIDs = make(map[string]bool)
	t.recoveredPartIDs = make(map[string]bool)

	sessions, err := client.GetSessionStatus(ctx)
	if err != nil {
		events = append(events, recoveryErrorEvent(t.nextSeq(), "session status query failed", err))
		return events
	}

	// Detect new sessions and query messages for known sessions.
	seen := make(map[string]bool)
	for _, info := range sessions {
		seen[info.ID] = true

		s, known := t.sessions[info.ID]
		if !known {
			// New session — emit compensation.
			s = &trackedSession{
				messages: make(map[string]bool),
				parts:    make(map[string]bool),
			}
			t.sessions[info.ID] = s
			events = append(events, sessionRecoveryEvent(t.nextSeq(), info))
		}

		// Query messages for this session to find missed items.
		msgsResp, err := client.GetSessionMessages(ctx, info.ID)
		if err != nil {
			events = append(events, recoveryErrorEvent(t.nextSeq(),
				"message history query failed for session "+info.ID, err))
			continue
		}

		for _, rawMsg := range msgsResp {
			msgID, partIDs := extractMessageIDs(rawMsg)
			if msgID == "" {
				continue
			}
			if !s.messages[msgID] {
				s.messages[msgID] = true
				t.recoveredMsgIDs[msgID] = true
				events = append(events, messageRecoveryEvent(t.nextSeq(), info.ID, msgID, rawMsg))
				// New message — record all parts without emitting individual events.
				for _, pid := range partIDs {
					if !s.parts[pid] {
						s.parts[pid] = true
					}
				}
			} else {
				// Known message — emit compensation for each missing part.
				for _, pid := range partIDs {
					if !s.parts[pid] {
						s.parts[pid] = true
						t.recoveredPartIDs[pid] = true
						events = append(events, partRecoveryEvent(t.nextSeq(), info.ID, msgID, pid, rawMsg))
					}
				}
			}
		}
	}

	// Emit recovery-complete marker.
	events = append(events, runtime.Event{
		Type:     CodeaEventRuntimeConnected,
		Sequence: t.nextSeq(),
		Metadata: map[string]string{"recovered": "true"},
	})

	return events
}

func (t *SessionTracker) nextSeq() int64 {
	t.seq++
	return t.seq
}

// extractMessageIDs extracts the message ID and part IDs from an OpenCode
// session message. The real OpenCode API wraps each message as
// {"info": {"id": ...}, "parts": [{"id": ...}]}. Returns empty string if the
// message ID cannot be extracted.
func extractMessageIDs(msg any) (messageID string, partIDs []string) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", nil
	}

	// Real OpenCode API shape: {info: {id, ...}, parts: [{id, ...}]}.
	var withInfo struct {
		Info struct {
			ID string `json:"id"`
		} `json:"info"`
		Parts []struct {
			ID string `json:"id"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(data, &withInfo); err == nil && withInfo.Info.ID != "" {
		for _, p := range withInfo.Parts {
			if p.ID != "" {
				partIDs = append(partIDs, p.ID)
			}
		}
		return withInfo.Info.ID, partIDs
	}

	// Fallback: flat shape {id, content: [{id}]} used by some tests.
	var raw struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || raw.ID == "" {
		return "", nil
	}
	var full struct {
		Content []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &full); err == nil {
		for _, c := range full.Content {
			if c.ID != "" {
				partIDs = append(partIDs, c.ID)
			}
		}
	}
	return raw.ID, partIDs
}

func recoveryErrorEvent(seq int64, msg string, cause error) runtime.Event {
	return runtime.Event{
		Type:     CodeaEventRuntimeError,
		Sequence: seq,
		Error:    runtime.NewRecoveryError("Recovery", msg, cause),
	}
}

func sessionRecoveryEvent(seq int64, info OpenCodeSessionV2Info) runtime.Event {
	return runtime.Event{
		Type:      CodeaEventSessionCreated,
		Sequence:  seq,
		SessionID: info.ID,
		CreatedAt: time.UnixMilli(int64(info.Time.Created)),
		Metadata:  map[string]string{"recovered": "true", "title": info.Title},
	}
}

func messageRecoveryEvent(seq int64, sessionID, messageID string, raw any) runtime.Event {
	rawJSON, _ := json.Marshal(raw)
	return runtime.Event{
		Type:      CodeaEventMessageUpdated,
		Sequence:  seq,
		SessionID: sessionID,
		MessageID: messageID,
		Metadata:  map[string]string{"recovered": "true"},
		Raw:       rawJSON,
	}
}

func partRecoveryEvent(seq int64, sessionID, messageID, partID string, raw any) runtime.Event {
	rawJSON, _ := json.Marshal(raw)
	return runtime.Event{
		Type:      CodeaEventPartUpdated,
		Sequence:  seq,
		SessionID: sessionID,
		MessageID: messageID,
		PartID:    partID,
		Metadata:  map[string]string{"recovered": "true"},
		Raw:       rawJSON,
	}
}

// recoveryEventWrapper marks an SSERawEvent as a synthetic recovery event so the
// adapter can unwrap it directly without going through MapEvent.
type recoveryEventWrapper struct {
	CodeaRecovery bool         `json:"__codea_recovery"`
	Event         runtime.Event `json:"event"`
}

func wrapRecoveryEvent(ev runtime.Event) ([]byte, error) {
	return json.Marshal(recoveryEventWrapper{CodeaRecovery: true, Event: ev})
}

func unwrapRecoveryEvent(data []byte) (runtime.Event, bool) {
	var w recoveryEventWrapper
	if err := json.Unmarshal(data, &w); err != nil || !w.CodeaRecovery {
		return runtime.Event{}, false
	}
	return w.Event, true
}

// MakeRecoveryHook returns a ReconnectHook that runs SessionTracker.Recover
// and converts the resulting runtime.Events to SSERawEvent for injection into
// the reconnected SSE stream.
func (t *SessionTracker) MakeRecoveryHook(client *HTTPClient) ReconnectHook {
	return func(ctx context.Context) ([]SSERawEvent, error) {
		events := t.Recover(ctx, client)
		raw := make([]SSERawEvent, 0, len(events))
		for _, ev := range events {
			data, err := wrapRecoveryEvent(ev)
			if err != nil {
				return nil, fmt.Errorf("wrap recovery event: %w", err)
			}
			raw = append(raw, SSERawEvent{Data: data})
		}
		return raw, nil
	}
}
