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
			raw:      `{"directory":"/tmp","payload":{"type":"permission.asked","properties":{"sessionID":"s1","id":"perm_abc","permission":"read"}}}`,
			wantType: CodeaEventApprovalRequested,
		},
		{
			name:     "permission.replied → approval.resolved",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.replied","properties":{"sessionID":"s1","requestID":"perm_abc","reply":"once"}}}`,
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

func TestEventMapperUsesNestedPartSessionIDWhenTopLevelMissing(t *testing.T) {
	raw := `{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"part":{"id":"p1","messageID":"m1","sessionID":"s-nested","type":"step-finish","reason":"stop"}}}}`
	event, err := MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.SessionID != "s-nested" {
		t.Fatalf("SessionID=%q, want nested part session id", event.SessionID)
	}
}

func TestEventMapperExtractsApprovalRequest(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantID   string
		wantPerm string
	}{
		{
			name:     "permission.asked with id and permission",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.asked","properties":{"sessionID":"s1","id":"perm_read_001","permission":"read"}}}`,
			wantID:   "perm_read_001",
			wantPerm: "read",
		},
		{
			name:     "permission.v2.asked with id and action",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.v2.asked","properties":{"sessionID":"s1","id":"perm_v2_002","action":"write"}}}`,
			wantID:   "perm_v2_002",
			wantPerm: "write",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := MapEvent([]byte(tt.raw), 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event.Type != CodeaEventApprovalRequested {
				t.Fatalf("expected approval.requested, got %q", event.Type)
			}
			if event.Approval == nil {
				t.Fatal("expected Approval to be non-nil")
			}
			if event.Approval.ID != tt.wantID {
				t.Fatalf("expected Approval.ID=%q, got %q", tt.wantID, event.Approval.ID)
			}
			if event.Approval.Permission != tt.wantPerm {
				t.Fatalf("expected Approval.Permission=%q, got %q", tt.wantPerm, event.Approval.Permission)
			}
		})
	}
}

func TestEventMapperExtractsApprovalCommand(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantCommand string
	}{
		{
			name:        "metadata.command preferred",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.asked","properties":{"id":"per_1","sessionID":"s1","permission":"bash","patterns":["rm -rf /"],"metadata":{"command":"rm -rf /build"}}}}`,
			wantCommand: "rm -rf /build",
		},
		{
			name:        "patterns fallback",
			raw:      `{"directory":"/tmp","payload":{"type":"permission.asked","properties":{"id":"per_2","sessionID":"s1","permission":"bash","patterns":["git","status"]}}}`,
			wantCommand: "git status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := MapEvent([]byte(tt.raw), 1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event.Approval == nil || event.Approval.Command != tt.wantCommand {
				t.Fatalf("Approval.Command = %q, want %q", event.Approval.Command, tt.wantCommand)
			}
		})
	}
}

// The remaining mapper tests are intentionally kept in this file below.
