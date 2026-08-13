package app

import (
	"context"
	"time"

	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// refreshInterval is the ~50ms merge-refresh cadence. It coalesces
// high-frequency streaming redraws and keeps live duration/status text moving.
const refreshInterval = 50 * time.Millisecond

// SubscribeEvents calls AgentRuntime.Subscribe and returns either the event
// channel (wrapped in subscribedMsg) or a subscribeErrMsg. It never blocks the
// Bubble Tea event loop.
func SubscribeEvents(client runtime.AgentRuntime) tea.Cmd {
	return func() tea.Msg {
		ch, err := client.Subscribe(context.Background())
		if err != nil {
			return subscribeErrMsg{err: err}
		}
		return subscribedMsg{ch: ch}
	}
}

// waitForEvent reads exactly one event from ch and wraps it in a message. The
// model re-issues it after each event so events are consumed one at a time and
// Update is never blocked on the channel.
func waitForEvent(ch <-chan runtime.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return eventStreamClosedMsg{}
		}
		return runtimeEventMsg{ev: ev}
	}
}

// TickCmd returns a command that fires once after refreshInterval.
func TickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{t: t}
	})
}

// PromptCmd sends a prompt to an existing session and reports the result. It
// never blocks the Bubble Tea event loop.
func PromptCmd(client runtime.AgentRuntime, sessionID runtime.SessionID, req runtime.PromptRequest) tea.Cmd {
	return func() tea.Msg {
		err := client.Prompt(context.Background(), sessionID, req)
		return promptResultMsg{sessionID: sessionID, err: err}
	}
}

// CreateSessionAndPromptCmd creates a session (using title) and sends req to it
// in one non-blocking command, returning the new session ID.
func CreateSessionAndPromptCmd(client runtime.AgentRuntime, title string, req runtime.PromptRequest) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		session, err := client.CreateSession(ctx, runtime.CreateSessionRequest{Title: title})
		if err != nil {
			return promptResultMsg{err: err}
		}
		sid := runtime.SessionID(session.ID)
		if err := client.Prompt(ctx, sid, req); err != nil {
			return promptResultMsg{sessionID: sid, err: err}
		}
		return promptResultMsg{sessionID: sid}
	}
}
