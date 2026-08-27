package app

import (
	"context"
	"time"

	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// refreshInterval is the 25ms merge-refresh cadence. It stays inside Task 25's
// 16–33ms bounded window, coalesces high-frequency streaming redraws, and keeps
// live lifecycle/status text responsive without repainting per token.
const refreshInterval = 25 * time.Millisecond

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

// CreateSessionCmd creates a session and returns its ID as a sessionCreatedMsg,
// before the first prompt is sent. Splitting creation from the prompt lets the
// model establish its current session ID before any of that session's events
// arrive, which session isolation (acceptsEvent) depends on.
func CreateSessionCmd(client runtime.AgentRuntime, title string) tea.Cmd {
	return func() tea.Msg {
		session, err := client.CreateSession(context.Background(), runtime.CreateSessionRequest{Title: title})
		if err != nil {
			return sessionCreatedMsg{err: err}
		}
		return sessionCreatedMsg{sessionID: runtime.SessionID(session.ID)}
	}
}

// ListSessionsCmd fetches the session list for the session panel. It never
// blocks the Bubble Tea event loop.
func ListSessionsCmd(client runtime.AgentRuntime) tea.Cmd {
	return func() tea.Msg {
		sessions, err := client.ListSessions(context.Background())
		return listSessionsResultMsg{sessions: sessions, err: err}
	}
}

// ListAgentsCmd fetches Codea-domain agents for the /agents workspace action.
func ListAgentsCmd(client runtime.AgentRuntime) tea.Cmd {
	return func() tea.Msg {
		agents, err := client.ListAgents(context.Background())
		return listAgentsResultMsg{agents: agents, err: err}
	}
}

// RuntimeHealthCmd reuses the existing AgentRuntime health contract for Task
// 22's /doctor quick check. Task 23 owns the shared full Doctor service.
func RuntimeHealthCmd(client runtime.AgentRuntime) tea.Cmd {
	return func() tea.Msg {
		health, err := client.Health(context.Background())
		return runtimeHealthResultMsg{health: health, err: err}
	}
}

// CancelResponseCmd cancels the active session response without bypassing the
// Codea-owned runtime contract.
func CancelResponseCmd(client runtime.AgentRuntime, sessionID runtime.SessionID) tea.Cmd {
	return func() tea.Msg {
		err := client.Cancel(context.Background(), sessionID)
		return cancelResponseResultMsg{sessionID: sessionID, err: err}
	}
}

// ReplyApprovalCmd sends an approval decision to the Runtime. It never blocks
// the Bubble Tea event loop; the result is delivered as approvalResultMsg.
func ReplyApprovalCmd(client runtime.AgentRuntime, approvalID runtime.ApprovalID, reply runtime.ApprovalReply) tea.Cmd {
	return func() tea.Msg {
		err := client.ReplyApproval(context.Background(), approvalID, reply)
		return approvalResultMsg{approvalID: approvalID, err: err}
	}
}

// LoadSessionHistoryCmd fetches a session's message history for resume. It never
// blocks the Bubble Tea event loop; the result is delivered as loadHistoryResultMsg.
func LoadSessionHistoryCmd(client runtime.AgentRuntime, sessionID runtime.SessionID) tea.Cmd {
	return func() tea.Msg {
		messages, err := client.GetSessionMessages(context.Background(), sessionID)
		return loadHistoryResultMsg{sessionID: sessionID, messages: messages, err: err}
	}
}
