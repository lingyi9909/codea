package app

import (
	"testing"

	"codea/tui/internal/reasoning"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// step.finished must finalize an in-progress reasoning block even when no
// answer delta arrived to implicitly end it, so the reasoning UI, duration, and
// assistant stream all reach a consistent terminal state.

func TestStepFinishedFinalizesReasoningWithoutAnswer(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "reasoning.delta", Content: "thinking..."}})
	if !m.reasoningActive {
		t.Fatal("precondition: reasoning should be active")
	}

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "step.finished"}})

	if m.reasoningActive {
		t.Error("reasoningActive = true, want false after step.finished")
	}
	snap := m.proc.Snapshot()
	if snap.HasActive() {
		t.Error("reasoning processor still has an active block after step.finished")
	}
	if len(snap.Blocks) != 1 || snap.Blocks[0].State != reasoning.BlockCompleted {
		t.Errorf("block state = %+v, want a single completed block", snap.Blocks)
	}
	if m.reasoningDuration <= 0 {
		t.Errorf("reasoningDuration = %v, want > 0 after step.finished", m.reasoningDuration)
	}
	if m.isStreaming {
		t.Error("isStreaming = true, want false after step.finished")
	}
	if !m.messages[1].Finished {
		t.Error("assistant message should be finished after step.finished")
	}
}

func TestSessionErrorKeepsInterruptedSemantics(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "reasoning.delta", Content: "thinking..."}})
	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "session.error"}})

	snap := m.proc.Snapshot()
	if len(snap.Blocks) != 1 || snap.Blocks[0].State != reasoning.BlockInterrupted {
		t.Errorf("block state = %+v, want a single interrupted block", snap.Blocks)
	}
	if m.reasoningActive {
		t.Error("reasoningActive = true, want false after session.error")
	}
}
