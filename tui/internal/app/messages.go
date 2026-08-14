package app

import (
	"time"

	"codea/tui/internal/runtime"
)

// subscribedMsg carries the event channel produced by a successful Subscribe.
// The model stores it and then issues waitForEvent to consume it one event at a
// time, never blocking the Bubble Tea event loop.
type subscribedMsg struct {
	ch <-chan runtime.Event
}

// runtimeEventMsg wraps a single consumed Runtime event.
type runtimeEventMsg struct {
	ev runtime.Event
}

// subscribeErrMsg reports a Subscribe failure. The TUI must not panic; it shows
// a degraded status and lets Task 4's recovery path handle reconnection.
type subscribeErrMsg struct {
	err error
}

// eventStreamClosedMsg signals the runtime event channel was closed.
type eventStreamClosedMsg struct{}

// tickMsg drives the ~50ms merge refresh used to coalesce high-frequency
// streaming redraws and keep live duration/status text moving.
type tickMsg struct {
	t time.Time
}

// runtimeStatusMsg updates the Runtime lifecycle status shown in the header.
type runtimeStatusMsg struct {
	status runtime.RuntimeStatus
}

// promptResultMsg reports the outcome of a prompt submission. On success it
// carries the session the prompt was sent to (created on first submit if none
// existed yet); on failure err is set and streaming is aborted.
type promptResultMsg struct {
	sessionID runtime.SessionID
	err       error
}

// sessionCreatedMsg reports a newly created session, delivered before the first
// prompt is sent so the model establishes its current session ID before any of
// that session's events can arrive (session isolation depends on knowing it).
type sessionCreatedMsg struct {
	sessionID runtime.SessionID
	err       error
}
