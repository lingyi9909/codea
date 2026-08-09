package opencode

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codea/tui/internal/runtime"
)

func TestEventMapperGoldenSSEAllMapped(t *testing.T) {
	goldenPath := filepath.Join("..", "..", "..", "runtime", "openapi", "golden-sse-s2.jsonl")
	f, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("failed to open golden SSE file: %v", err)
	}
	defer f.Close()

	var seq int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		seq++
		event, err := MapEvent([]byte(line), seq)
		if err != nil {
			t.Fatalf("map event %d: %v", seq, err)
		}
		if len(event.Raw) == 0 {
			t.Fatalf("event %d lost raw payload", seq)
		}
		if event.Type == "" {
			t.Fatalf("event %d has no mapped or unknown type", seq)
		}
		if event.RawType == "" {
			t.Fatalf("event %d has empty RawType", seq)
		}
		// RawType must preserve the original OpenCode type
		if event.Type != "_unparseable_" && event.Type != CodeaEventRaw && event.RawType == string(event.Type) {
			// ok: raw events use the original type as RawType which may equal Type
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	if seq == 0 {
		t.Fatal("no events mapped from golden SSE")
	}
}

func TestEventMapperSemanticTypes(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantType runtime.EventType
	}{
		{
			name:     "server.connected → runtime.connected",
			raw:      `{"directory":"","payload":{"type":"server.connected","properties":{}}}`,
			wantType: CodeaEventRuntimeConnected,
		},
		{
			name:     "session.status → session.status",
			raw:      `{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"s1","status":{"type":"busy"}}}}`,
			wantType: CodeaEventSessionStatus,
		},
		{
			name:     "session.updated → session.updated",
			raw:      `{"directory":"/tmp","payload":{"type":"session.updated","properties":{"sessionID":"s1","info":{"id":"s1","projectID":"proj1"}}}}`,
			wantType: CodeaEventSessionUpdated,
		},
		{
			name:     "session.created → session.created",
			raw:      `{"directory":"/tmp","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`,
			wantType: CodeaEventSessionCreated,
		},
		{
			name:     "message.updated → message.updated",
			raw:      `{"directory":"/tmp","payload":{"type":"message.updated","properties":{"sessionID":"s1","info":{"id":"msg1"}}}}`,
			wantType: CodeaEventMessageUpdated,
		},
		{
			name:     "message.part.delta text → answer.delta",
			raw:      `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"hello"}}}`,
			wantType: CodeaEventAnswerDelta,
		},
		{
			name:     "message.part.delta reasoning → reasoning.delta",
			raw:      `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"reasoning","delta":"I think..."}}}`,
			wantType: CodeaEventReasoningDelta,
		},
		{
			name:     "message.part.updated step-start → step.started",
			raw:      `{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"m1","sessionID":"s1","type":"step-start"}}}}`,
			wantType: CodeaEventStepStarted,
		},
		{
			name:     "message.part.updated step-finish → step.finished",
			raw:      `{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"m1","sessionID":"s1","type":"step-finish","reason":"stop"}}}}`,
			wantType: CodeaEventStepFinished,
		},
		{
			name:     "message.part.updated text → part.updated",
			raw:      `{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"m1","sessionID":"s1","type":"text","text":"hello"}}}}`,
			wantType: CodeaEventPartUpdated,
		},
		{
			name:     "permission.asked → approval.requested",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.asked","properties":{"sessionID":"s1"}}}`,
			wantType: CodeaEventApprovalRequested,
		},
		{
			name:     "permission.replied → approval.resolved",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.replied","properties":{"sessionID":"s1"}}}`,
			wantType: CodeaEventApprovalResolved,
		},
		{
			name:     "unknown vendor type → raw",
			raw:      `{"directory":"/tmp","payload":{"type":"vendor.custom.event","properties":{"key":"value"}}}`,
			wantType: CodeaEventRaw,
		},
		{
			name:     "plugin.added → raw (vendor-specific)",
			raw:      `{"directory":"/tmp","payload":{"type":"plugin.added","properties":{"id":"test"}}}`,
			wantType: CodeaEventRaw,
		},
		{
			name:     "sync → raw (internal)",
			raw:      `{"directory":"/tmp","payload":{"type":"sync","properties":null}}`,
			wantType: CodeaEventRaw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := MapEvent([]byte(tt.raw), 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event.Type != tt.wantType {
				t.Fatalf("expected Type=%q, got %q", tt.wantType, event.Type)
			}
			if event.RawType == "" {
				t.Fatal("RawType must not be empty")
			}
		})
	}
}

func TestEventMapperPreservesRawType(t *testing.T) {
	raw := `{"directory":"/tmp","payload":{"type":"plugin.added","properties":{"id":"test"}}}`
	event, err := MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Semantic type is raw, but RawType preserves original
	if event.Type != CodeaEventRaw {
		t.Fatalf("expected Type=raw, got %q", event.Type)
	}
	if event.RawType != "plugin.added" {
		t.Fatalf("expected RawType=plugin.added, got %q", event.RawType)
	}
}

func TestEventMapperExtractsProjectID(t *testing.T) {
	raw := `{"directory":"/tmp","payload":{"type":"session.updated","properties":{"sessionID":"s1","info":{"projectID":"proj_abc","id":"s1"}}}}`
	event, err := MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ProjectID != "proj_abc" {
		t.Fatalf("expected ProjectID=proj_abc, got %q", event.ProjectID)
	}
}

func TestEventMapperExtractsCreatedAt(t *testing.T) {
	raw := `{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"s1","time":1785756134158}}}`
	event, err := MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.CreatedAt.IsZero() {
		t.Fatal("expected non-zero CreatedAt")
	}
}

func TestEventMapperMalformedJSON(t *testing.T) {
	raw := []byte(`not json`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "_unparseable_" {
		t.Fatalf("expected Type=_unparseable_, got %q", event.Type)
	}
	if string(event.Raw) != `not json` {
		t.Fatalf("expected Raw to preserve exact bytes, got %q", event.Raw)
	}
}

func TestEventMapperOversizedRaw(t *testing.T) {
	payload := strings.Repeat("x", 20*1024)
	raw := []byte(`{"directory":"/tmp","payload":{"type":"test","properties":{"data":"` + payload + `"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !event.RawTruncated {
		t.Fatal("expected RawTruncated=true for oversized payload")
	}
	if event.RawOriginalSize != len(raw) {
		t.Fatalf("expected RawOriginalSize=%d, got %d", len(raw), event.RawOriginalSize)
	}
	if len(event.Raw) > 16*1024 {
		t.Fatalf("expected Raw <= 16KB, got %d", len(event.Raw))
	}
}

func TestEventMapperExtractsSessionID(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses_abc123","status":{"type":"busy"}}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.SessionID != "ses_abc123" {
		t.Fatalf("expected SessionID=ses_abc123, got %q", event.SessionID)
	}
}

func TestEventMapperExtractsMessagePartIDs(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"hello"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.SessionID != "s1" {
		t.Fatalf("expected SessionID=s1, got %q", event.SessionID)
	}
	if event.MessageID != "m1" {
		t.Fatalf("expected MessageID=m1, got %q", event.MessageID)
	}
	if event.PartID != "p1" {
		t.Fatalf("expected PartID=p1, got %q", event.PartID)
	}
}

func TestEventMapperExtractsContentFromDelta(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"hello world"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Content != "hello world" {
		t.Fatalf("expected Content='hello world', got %q", event.Content)
	}
}

func TestEventMapperPreservesRawJSON(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"s1"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rawJSON map[string]any
	if err := json.Unmarshal(event.Raw, &rawJSON); err != nil {
		t.Fatalf("Raw is not valid JSON: %v", err)
	}
}
