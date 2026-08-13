package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestWaitForEventReadsNextEvent(t *testing.T) {
	ch := make(chan runtime.Event, 1)
	ch <- runtime.Event{Type: runtime.EventType("answer.delta"), Content: "hello"}

	msg := waitForEvent(ch)()
	got, ok := msg.(runtimeEventMsg)
	if !ok {
		t.Fatalf("waitForEvent returned %T, want runtimeEventMsg", msg)
	}
	if got.ev.Content != "hello" {
		t.Errorf("event content = %q, want %q", got.ev.Content, "hello")
	}
}

func TestWaitForEventClosedChannel(t *testing.T) {
	ch := make(chan runtime.Event)
	close(ch)

	msg := waitForEvent(ch)()
	if _, ok := msg.(eventStreamClosedMsg); !ok {
		t.Fatalf("waitForEvent on closed channel returned %T, want eventStreamClosedMsg", msg)
	}
}

func TestSubscribeEventsSuccess(t *testing.T) {
	client := fakeruntime.New()
	msg := SubscribeEvents(client)()
	sm, ok := msg.(subscribedMsg)
	if !ok {
		t.Fatalf("SubscribeEvents returned %T, want subscribedMsg", msg)
	}
	if sm.ch == nil {
		t.Error("subscribedMsg channel is nil")
	}
}

func TestSubscribeEventsError(t *testing.T) {
	client := fakeruntime.New()
	client.SubscribeError = fakeruntime.ErrSimulated

	msg := SubscribeEvents(client)()
	se, ok := msg.(subscribeErrMsg)
	if !ok {
		t.Fatalf("SubscribeEvents returned %T, want subscribeErrMsg", msg)
	}
	if se.err == nil {
		t.Error("subscribeErrMsg err is nil")
	}
}
