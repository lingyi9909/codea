package app

import (
	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// Init starts the runtime subscription and the merge-refresh ticker.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(SubscribeEvents(m.runtimeClient), TickCmd())
}

// Update handles Bubble Tea messages. Subscription lifecycle is wired here;
// streaming prompt/answer processing is added in later steps.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case subscribedMsg:
		m.eventCh = msg.ch
		return m, waitForEvent(msg.ch)

	case runtimeEventMsg:
		if m.eventCh != nil {
			return m, waitForEvent(m.eventCh)
		}
		return m, nil

	case subscribeErrMsg:
		m.runtimeStatus = runtime.RuntimeCrashed
		return m, nil

	case eventStreamClosedMsg:
		if m.runtimeStatus != runtime.RuntimeCrashed {
			m.runtimeStatus = runtime.RuntimeStopped
		}
		return m, nil

	case tickMsg:
		return m, TickCmd()
	}

	return m, nil
}
