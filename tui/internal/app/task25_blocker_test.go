package app

import (
	"strings"
	"testing"
	"time"
)

func TestTask25SecretRedactionTerminatesAndRedactsStableMarker(t *testing.T) {
	done := make(chan string, 1)
	go func() {
		done <- redactCommonSecret("token=supersecret")
	}()

	select {
	case got := <-done:
		if strings.Contains(got, "supersecret") {
			t.Fatalf("secret redaction leaked value: %q", got)
		}
		if got != "token=***" {
			t.Fatalf("secret redaction = %q, want token=***", got)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("secret redaction did not terminate for a marker that remains after replacement")
	}
}

func TestTask25FocusSummaryPrefersActiveTurnOverPreviousCompletedTurn(t *testing.T) {
	m := NewModel(nil)
	start := time.Date(2026, 8, 27, 6, 0, 0, 0, time.UTC)

	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceWorking,
		Title:         "Working",
		Status:        TraceSuccess,
		InvocationKey: "turn:old:working",
		StartedAt:     start,
		FinishedAt:    start.Add(time.Second),
		TurnID:        "old",
	})
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceTool, Title: "read", Status: TraceSuccess, InvocationKey: "tool:old-1", TurnID: "old"})
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceTool, Title: "grep", Status: TraceSuccess, InvocationKey: "tool:old-2", TurnID: "old"})

	m.activeTurnID = "current"
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceWorking,
		Title:         "Working",
		Status:        TraceRunning,
		InvocationKey: "turn:current:working",
		StartedAt:     start.Add(2 * time.Second),
		TurnID:        "current",
	})
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceSkill, Title: "debug", Status: TraceRunning, InvocationKey: "skill:current", TurnID: "current"})

	got := m.renderFocusActivitySummary()
	if !strings.Contains(got, "1 skill") {
		t.Fatalf("focus summary = %q, want current turn activity", got)
	}
	if strings.Contains(got, "2 tool calls") {
		t.Fatalf("focus summary leaked previous completed turn activity: %q", got)
	}
}
