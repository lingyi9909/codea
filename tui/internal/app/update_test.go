package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestUpdateSubscribedMsgStoresChannel(t *testing.T) {
	m := NewModel(fakeruntime.New())
	ch := make(chan runtime.Event)

	_, cmd := m.Update(subscribedMsg{ch: ch})

	if m.eventCh != ch {
		t.Error("eventCh not stored")
	}
	if cmd == nil {
		t.Error("subscribedMsg should issue a waitForEvent cmd")
	}
}

func TestUpdateSubscribeErrorMarksCrashed(t *testing.T) {
	m := NewModel(fakeruntime.New())

	m.Update(subscribeErrMsg{err: fakeruntime.ErrSimulated})

	if m.runtimeStatus != runtime.RuntimeCrashed {
		t.Errorf("runtimeStatus = %q, want crashed", m.runtimeStatus)
	}
}

func TestUpdateEventStreamClosedMarksStopped(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.runtimeStatus = runtime.RuntimeHealthy

	m.Update(eventStreamClosedMsg{})

	if m.runtimeStatus != runtime.RuntimeStopped {
		t.Errorf("runtimeStatus = %q, want stopped", m.runtimeStatus)
	}
}

func TestUpdateTickRearms(t *testing.T) {
	m := NewModel(fakeruntime.New())

	_, cmd := m.Update(tickMsg{})

	if cmd == nil {
		t.Error("tickMsg should re-arm TickCmd")
	}
}

func TestUpdateRuntimeEventContinuesConsuming(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.eventCh = make(chan runtime.Event)

	_, cmd := m.Update(runtimeEventMsg{ev: runtime.Event{Type: runtime.EventType("answer.delta")}})

	if cmd == nil {
		t.Error("runtimeEventMsg should issue a waitForEvent cmd to keep consuming")
	}
}
