package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func TestSessionTrackerRecord(t *testing.T) {
	tracker := NewSessionTracker()

	ev := runtime.Event{
		Sequence:  5,
		SessionID: "s1",
		MessageID: "m1",
		PartID:    "p1",
	}
	tracker.Record(ev)

	if tracker.LastSeq() != 5 {
		t.Errorf("LastSeq = %d, want 5", tracker.LastSeq())
	}

	tracker.mu.Lock()
	s := tracker.sessions["s1"]
	tracker.mu.Unlock()
	if s == nil {
		t.Fatal("session s1 not tracked")
	}
	if !s.messages["m1"] {
		t.Error("message m1 not tracked")
	}
	if !s.parts["p1"] {
		t.Error("part p1 not tracked")
	}
}

func TestSessionTrackerLastSeqMonotonic(t *testing.T) {
	tracker := NewSessionTracker()

	tracker.Record(runtime.Event{Sequence: 3, SessionID: "a"})
	tracker.Record(runtime.Event{Sequence: 1, SessionID: "b"})

	if tracker.LastSeq() != 3 {
		t.Errorf("LastSeq = %d, want 3", tracker.LastSeq())
	}
}

func TestSessionTrackerSkipsEmptySession(t *testing.T) {
	tracker := NewSessionTracker()

	tracker.Record(runtime.Event{Sequence: 1})

	tracker.mu.Lock()
	n := len(tracker.sessions)
	tracker.mu.Unlock()
	if n != 0 {
		t.Errorf("expected 0 sessions, got %d", n)
	}
}

func TestExtractMessageIDs(t *testing.T) {
	// Assistant message with content parts.
	msg := map[string]any{
		"id":   "msg-1",
		"type": "assistant",
		"content": []map[string]any{
			{"id": "part-1", "type": "text"},
			{"id": "part-2", "type": "tool"},
		},
	}
	msgID, partIDs := extractMessageIDs(msg)
	if msgID != "msg-1" {
		t.Errorf("msgID = %q, want msg-1", msgID)
	}
	if len(partIDs) != 2 || partIDs[0] != "part-1" || partIDs[1] != "part-2" {
		t.Errorf("partIDs = %v, want [part-1 part-2]", partIDs)
	}
}

func TestExtractMessageIDsNoID(t *testing.T) {
	msg := map[string]any{
		"type": "assistant",
	}
	msgID, _ := extractMessageIDs(msg)
	if msgID != "" {
		t.Errorf("msgID = %q, want empty", msgID)
	}
}

func TestExtractMessageIDsNoContent(t *testing.T) {
	msg := map[string]any{
		"id":   "msg-1",
		"type": "system",
	}
	msgID, partIDs := extractMessageIDs(msg)
	if msgID != "msg-1" {
		t.Errorf("msgID = %q, want msg-1", msgID)
	}
	if len(partIDs) != 0 {
		t.Errorf("partIDs = %v, want empty", partIDs)
	}
}

func TestExtractMessageIDsRealAPIShape(t *testing.T) {
	// Real OpenCode /session/:id/message response wraps each message as
	// {"info": {"id": ...}, "parts": [{"id": ...}]}.
	msg := map[string]any{
		"info": map[string]any{"id": "msg-real", "role": "assistant"},
		"parts": []map[string]any{
			{"id": "part-real-1", "type": "text"},
			{"id": "part-real-2", "type": "tool"},
		},
	}
	msgID, partIDs := extractMessageIDs(msg)
	if msgID != "msg-real" {
		t.Errorf("msgID = %q, want msg-real", msgID)
	}
	if len(partIDs) != 2 || partIDs[0] != "part-real-1" || partIDs[1] != "part-real-2" {
		t.Errorf("partIDs = %v, want [part-real-1 part-real-2]", partIDs)
	}
}

func TestRecoveryDetectsNewSession(t *testing.T) {
	var statusCalls atomic.Int32
	var msgCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/session" {
			statusCalls.Add(1)
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{
				{ID: "s1", Title: "Test Session", Time: OpenCodeSessionV2InfoTime{Created: 1000}},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/message") {
			msgCalls.Add(1)
			json.NewEncoder(w).Encode([]OpenCodeSessionMessage{
				map[string]any{"id": "m1", "type": "assistant", "content": []any{}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "")
	tracker := NewSessionTracker()

	events := tracker.Recover(context.Background(), client)
	if len(events) < 3 {
		t.Fatalf("expected at least 3 recovery events (session + message + connected), got %d", len(events))
	}

	// First event should be session.created.
	if events[0].Type != CodeaEventSessionCreated {
		t.Errorf("event[0].Type = %s, want %s", events[0].Type, CodeaEventSessionCreated)
	}
	if events[0].SessionID != "s1" {
		t.Errorf("event[0].SessionID = %s, want s1", events[0].SessionID)
	}

	// Second event should be message.updated.
	if events[1].Type != CodeaEventMessageUpdated {
		t.Errorf("event[1].Type = %s, want %s", events[1].Type, CodeaEventMessageUpdated)
	}
	if events[1].MessageID != "m1" {
		t.Errorf("event[1].MessageID = %s, want m1", events[1].MessageID)
	}

	// Last event should be the recovery-complete marker.
	last := events[len(events)-1]
	if last.Type != CodeaEventRuntimeConnected {
		t.Errorf("last event type = %s, want %s", last.Type, CodeaEventRuntimeConnected)
	}

	if statusCalls.Load() != 1 {
		t.Errorf("statusCalls = %d, want 1", statusCalls.Load())
	}
}

func TestRecoverySkipsKnownMessages(t *testing.T) {
	var msgCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/session" {
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{
				{ID: "s1", Time: OpenCodeSessionV2InfoTime{Created: 1000}},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/message") {
			msgCalls.Add(1)
			json.NewEncoder(w).Encode([]OpenCodeSessionMessage{
				map[string]any{"id": "m1", "type": "assistant", "content": []any{}},
				map[string]any{"id": "m2", "type": "assistant", "content": []any{}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "")
	tracker := NewSessionTracker()

	// Pre-record m1 as already seen.
	tracker.Record(runtime.Event{SessionID: "s1", MessageID: "m1"})

	events := tracker.Recover(context.Background(), client)

	// Should only emit m2 (not m1) + session + connected.
	msgCount := 0
	for _, ev := range events {
		if ev.Type == CodeaEventMessageUpdated {
			msgCount++
			if ev.MessageID != "m2" {
				t.Errorf("unexpected message %s", ev.MessageID)
			}
		}
	}
	if msgCount != 1 {
		t.Errorf("expected 1 new message event, got %d", msgCount)
	}
}

func TestRecoveryCompensatesMissingPart(t *testing.T) {
	// Known message m1 with known part p1. History now has m1 with p1 + new p2.
	// Recovery must emit a part.updated event for p2 only (not p1).

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/session" {
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{
				{ID: "s1", Time: OpenCodeSessionV2InfoTime{Created: 1000}},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/message") {
			json.NewEncoder(w).Encode([]OpenCodeSessionMessage{
				map[string]any{
					"id":   "m1",
					"type": "assistant",
					"content": []map[string]any{
						{"id": "part-1", "type": "text"},
						{"id": "part-2", "type": "text"},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "")
	tracker := NewSessionTracker()

	// Pre-record: m1 is known, part-1 is known.
	tracker.Record(runtime.Event{SessionID: "s1", MessageID: "m1", PartID: "part-1"})

	events := tracker.Recover(context.Background(), client)

	// Should have: session recovery + part compensation (part-2 only) + connected marker.
	var partEvents []runtime.Event
	for _, ev := range events {
		if ev.Type == CodeaEventPartUpdated {
			partEvents = append(partEvents, ev)
		}
	}
	if len(partEvents) != 1 {
		t.Fatalf("expected 1 part compensation event, got %d", len(partEvents))
	}
	if partEvents[0].PartID != "part-2" {
		t.Errorf("expected part-2, got %s", partEvents[0].PartID)
	}
	if partEvents[0].Metadata["recovered"] != "true" {
		t.Error("part compensation must have recovered=true metadata")
	}

	// Verify no duplicate part-1 events.
	for _, ev := range partEvents {
		if ev.PartID == "part-1" {
			t.Error("part-1 should NOT be emitted (already known)")
		}
	}
}

func TestRecoveryDedupesKnownPart(t *testing.T) {
	// Known m1 with known p1. History still has only m1/p1 — no new parts.
	// Recovery must emit zero part events (no duplicates).

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/session" {
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{
				{ID: "s1", Time: OpenCodeSessionV2InfoTime{Created: 1000}},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/message") {
			json.NewEncoder(w).Encode([]OpenCodeSessionMessage{
				map[string]any{
					"id":   "m1",
					"type": "assistant",
					"content": []map[string]any{
						{"id": "part-1", "type": "text"},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "")
	tracker := NewSessionTracker()

	// Pre-record: m1 and part-1 are both known.
	tracker.Record(runtime.Event{SessionID: "s1", MessageID: "m1", PartID: "part-1"})

	events := tracker.Recover(context.Background(), client)

	// Zero part compensation events.
	for _, ev := range events {
		if ev.Type == CodeaEventPartUpdated {
			t.Errorf("unexpected part event for already-known part: %s", ev.PartID)
		}
	}
}

func TestRecoverySessionStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewHTTPClient(srv.URL, "", "")
	tracker := NewSessionTracker()

	events := tracker.Recover(context.Background(), client)
	if len(events) != 1 {
		t.Fatalf("expected 1 error event, got %d", len(events))
	}
	if events[0].Type != CodeaEventRuntimeError {
		t.Errorf("expected runtime.error, got %s", events[0].Type)
	}
	if events[0].Error == nil || events[0].Error.Kind != runtime.RuntimeErrorRecovery {
		t.Error("expected Recovery error kind")
	}
}

func TestAdapterSubscribeUsesReconnectingClient(t *testing.T) {
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"payload\":{\"type\":\"answer.delta\",\"properties\":{\"content\":\"hello\"}}}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var disconnectCount int
	var eventCount int
	timeout := time.After(6 * time.Second)
loop:
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				break loop
			}
			eventCount++
			if ev.Type == CodeaEventRuntimeError && ev.Error != nil && ev.Error.Code == "DISCONNECTED" {
				disconnectCount++
			}
		case <-timeout:
			cancel()
		}
	}

	if connects.Load() < 2 {
		t.Errorf("expected at least 2 connections, got %d", connects.Load())
	}
	if disconnectCount < 1 {
		t.Errorf("expected at least 1 disconnect event, got %d", disconnectCount)
	}
	t.Logf("events: %d, connects: %d, disconnects: %d", eventCount, connects.Load(), disconnectCount)
}
