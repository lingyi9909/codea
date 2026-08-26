package app

import (
	"time"

	"codea/tui/internal/runtime"
)

// subscribedMsg carries the event channel produced by a successful Subscribe.
type subscribedMsg struct {
	ch <-chan runtime.Event
}

type runtimeEventMsg struct {
	ev runtime.Event
}

type subscribeErrMsg struct {
	err error
}

type eventStreamClosedMsg struct{}

type tickMsg struct {
	t time.Time
}

type runtimeStatusMsg struct {
	status runtime.RuntimeStatus
}

type promptResultMsg struct {
	sessionID runtime.SessionID
	err       error
}

type sessionCreatedMsg struct {
	sessionID runtime.SessionID
	err       error
}

type listSessionsResultMsg struct {
	sessions []runtime.Session
	err      error
}

// Task 22 command-workspace results remain Codea-domain only.
type listAgentsResultMsg struct {
	agents []runtime.Agent
	err    error
}

type runtimeHealthResultMsg struct {
	health runtime.HealthInfo
	err    error
}

type cancelResponseResultMsg struct {
	sessionID runtime.SessionID
	err       error
}

type approvalResultMsg struct {
	approvalID runtime.ApprovalID
	err        error
}

type loadHistoryResultMsg struct {
	sessionID runtime.SessionID
	messages  []runtime.Message
	err       error
}
