package opencode

import (
	"context"
	"encoding/json"
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
}

type trackedSession struct {
	messages map[string]bool
	parts    map[string]bool
}

// NewSessionTracker creates a new tracker.
func NewSessionTracker() *SessionTracker {
	return &SessionTracker{
		sessions: make(map[string]*trackedSession),
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

// Recover queries the OpenCode API for current session/message state and
// returns compensation events for any sessions or messages that are new
// since the last observation.
func (t *SessionTracker) Recover(ctx context.Context, client *HTTPClient) []runtime.Event {
	t.mu.Lock()
	defer t.mu.Unlock()

	var events []runtime.Event

	sessionsResp, err := client.GetSessionStatus(ctx)
	if err != nil {
		events = append(events, recoveryErrorEvent(t.nextSeq(), "session status query failed", err))
		return events
	}

	// Detect new sessions and query messages for known sessions.
	seen := make(map[string]bool)
	for _, info := range sessionsResp.Data {
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

		for _, rawMsg := range msgsResp.Data {
			msgID, partIDs := extractMessageIDs(rawMsg)
			if msgID == "" {
				continue
			}
			if !s.messages[msgID] {
				s.messages[msgID] = true
				events = append(events, messageRecoveryEvent(t.nextSeq(), info.ID, msgID, rawMsg))
			}
			for _, pid := range partIDs {
				if !s.parts[pid] {
					s.parts[pid] = true
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
// session message (which is a tagged union). Returns empty string if ID
// cannot be extracted.
func extractMessageIDs(msg any) (messageID string, partIDs []string) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", nil
	}
	var raw struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || raw.ID == "" {
		return "", nil
	}
	// Extract part IDs from content array for assistant/tool messages.
	var content []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	var full struct {
		Content []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &full); err == nil {
		content = full.Content
	}
	for _, c := range content {
		if c.ID != "" {
			partIDs = append(partIDs, c.ID)
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
