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
