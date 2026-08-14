package parity_test

import (
	"fmt"
	"testing"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

// TestGeneralAgentToolLifecycleCorrelation proves the General Agent tool
// lifecycle is observable end-to-end through the OpenCode mapper: tool.called
// → tool.success → assistant continues, with the same CallID correlating the
// lifecycle. This is the "tool.called must not exist while tool.success is
// lost" guarantee from Task 9.
func TestGeneralAgentToolLifecycleCorrelation(t *testing.T) {
	rawEvents := []string{
		// tool.called via the legacy message.part.updated path.
		`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"m1","sessionID":"s1","type":"tool","tool":"read","callID":"call-read-1"}}}}`,
		// tool.success via the modern session.next.tool.* path.
		`{"directory":"/tmp","payload":{"type":"session.next.tool.success","properties":{"sessionID":"s1","callID":"call-read-1"}}}`,
		// assistant continues with an answer.
		`{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p2","field":"text","delta":"file contents"}}}`,
		// step completes.
		`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p3","messageID":"m1","sessionID":"s1","type":"step-finish"}}}}`,
	}

	var events []runtime.Event
	for i, raw := range rawEvents {
		ev, err := opencode.MapEvent([]byte(raw), int64(i+1))
		if err != nil {
			t.Fatalf("map event %d: %v", i, err)
		}
		events = append(events, ev)
	}

	var calledCallID, successCallID string
	var sawAnswer, sawStepFinish bool
	for _, ev := range events {
		switch ev.Type {
		case runtime.EventType("tool.called"):
			if ev.Tool == nil {
				t.Fatal("tool.called has nil Tool")
			}
			calledCallID = ev.Tool.CallID
		case runtime.EventType("tool.success"):
			if ev.Tool == nil {
				t.Fatal("tool.success has nil Tool")
			}
			successCallID = ev.Tool.CallID
		case runtime.EventType("answer.delta"):
			if ev.Content != "" {
				sawAnswer = true
			}
		case runtime.EventType("step.finished"):
			sawStepFinish = true
		}
	}

	if calledCallID == "" {
		t.Fatal("no tool.called event observed")
	}
	if successCallID == "" {
		t.Fatal("no tool.success event observed — tool lifecycle truncated")
	}
	if calledCallID != successCallID {
		t.Fatalf("CallID not correlated: called=%q success=%q", calledCallID, successCallID)
	}
	if !sawAnswer {
		t.Error("assistant did not continue with an answer after the tool")
	}
	if !sawStepFinish {
		t.Error("step did not finish after the tool lifecycle")
	}
}

// TestGeneralAgentToolFailedCorrelation proves a failed tool is still
// observable with the same CallID, so the Application can mark it failed.
func TestGeneralAgentToolFailedCorrelation(t *testing.T) {
	rawEvents := []string{
		`{"directory":"/tmp","payload":{"type":"session.next.tool.called","properties":{"sessionID":"s1","callID":"call-1","tool":"bash"}}}`,
		`{"directory":"/tmp","payload":{"type":"session.next.tool.failed","properties":{"sessionID":"s1","callID":"call-1"}}}`,
	}

	var calledCallID, failedCallID string
	for i, raw := range rawEvents {
		ev, err := opencode.MapEvent([]byte(raw), int64(i+1))
		if err != nil {
			t.Fatalf("map event %d: %v", i, err)
		}
		if ev.Tool == nil {
			t.Fatalf("event %d has nil Tool", i)
		}
		switch ev.Type {
		case runtime.EventType("tool.called"):
			calledCallID = ev.Tool.CallID
		case runtime.EventType("tool.failed"):
			failedCallID = ev.Tool.CallID
		}
	}
	if calledCallID == "" || failedCallID == "" {
		t.Fatalf("incomplete tool lifecycle: called=%q failed=%q", calledCallID, failedCallID)
	}
	if calledCallID != failedCallID {
		t.Fatalf("CallID not correlated: called=%q failed=%q", calledCallID, failedCallID)
	}
}

// TestGeneralAgentNativeToolNames proves each concrete native tool name passes
// through the mapper as a tool.called event with the correct name — Codea does
// not rename, drop, or otherwise mangle the native tool surface.
func TestGeneralAgentNativeToolNames(t *testing.T) {
	tools := []string{"read", "grep", "glob", "write", "edit", "bash"}
	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			raw := fmt.Sprintf(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"m1","sessionID":"s1","type":"tool","tool":"%s","callID":"call-%s"}}}}`, toolName, toolName)
			ev, err := opencode.MapEvent([]byte(raw), 1)
			if err != nil {
				t.Fatalf("map %s: %v", toolName, err)
			}
			if ev.Type != runtime.EventType("tool.called") {
				t.Fatalf("expected tool.called, got %q", ev.Type)
			}
			if ev.Tool == nil || ev.Tool.Name != toolName {
				t.Fatalf("expected Tool.Name=%q, got %+v", toolName, ev.Tool)
			}
			if ev.Tool.CallID != "call-"+toolName {
				t.Fatalf("expected Tool.CallID=call-%s, got %q", toolName, ev.Tool.CallID)
			}
		})
	}
}
