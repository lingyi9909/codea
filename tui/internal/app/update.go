package app

import (
	"fmt"
	"strings"

	"codea/tui/internal/components"
	"codea/tui/internal/reasoning"
	"codea/tui/internal/runtime"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Init starts the runtime subscription and the merge-refresh ticker.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(SubscribeEvents(m.runtimeClient), TickCmd())
}

// Update handles Bubble Tea messages: subscription lifecycle, key input,
// prompt submission, and streaming runtime event processing.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case subscribedMsg:
		m.eventCh = msg.ch
		m.runtimeStatus = runtime.RuntimeHealthy
		m.markDirty()
		return m, waitForEvent(msg.ch)

	case runtimeEventMsg:
		if m.processRuntimeEvent(msg.ev) {
			m.markDirty()
		}
		if m.eventCh != nil {
			return m, waitForEvent(m.eventCh)
		}
		return m, nil

	case promptResultMsg:
		if msg.err != nil {
			m.finishStreaming()
			m.markDirty()
			return m, nil
		}
		if msg.sessionID != "" {
			m.sessionID = msg.sessionID
		}
		return m, nil

	case sessionCreatedMsg:
		if msg.err != nil {
			m.pendingPrompt = nil
			m.finishStreaming()
			m.markDirty()
			return m, nil
		}
		m.sessionID = msg.sessionID
		req := m.pendingPrompt
		m.pendingPrompt = nil
		if req == nil {
			return m, nil
		}
		return m, PromptCmd(m.runtimeClient, m.sessionID, *req)

	case listSessionsResultMsg:
		if msg.err != nil {
			m.sessionNotice = "Failed to load sessions: " + msg.err.Error()
			m.markDirty()
			return m, nil
		}
		m.sessionPanel.Open(sessionItems(msg.sessions))
		m.sessionPanel.SetActive(m.sessionID)
		m.sessionNotice = ""
		m.markDirty()
		return m, nil

	case loadHistoryResultMsg:
		// A result only applies to the resume currently in flight. A stale result
		// (the panel was closed or a different resume started) is ignored.
		if m.pendingResumeID == "" || msg.sessionID != m.pendingResumeID {
			return m, nil
		}
		m.pendingResumeID = ""
		if msg.err != nil {
			// Failure does not switch sessions; the panel stays open with a notice
			// so the user can retry or pick another session.
			m.sessionNotice = "Failed to load session history: " + msg.err.Error()
			m.markDirty()
			return m, nil
		}
		m.resumeSession(msg.sessionID, msg.messages)
		m.sessionPanel.Close()
		m.markDirty()
		return m, nil

	case approvalResultMsg:
		// A result only applies to the approval currently shown. A stale result
		// (from a reply issued against a request that has since been superseded
		// by a new permission.asked) must not close or corrupt the current modal.
		if m.permission.Request == nil || runtime.ApprovalID(m.permission.Request.ID) != msg.approvalID {
			return m, nil
		}
		m.approvalPending = false
		if msg.err != nil {
			m.approvalErr = msg.err.Error()
			m.markDirty()
			return m, nil
		}
		m.permission = components.PermissionModel{}
		m.approvalErr = ""
		m.markDirty()
		return m, nil

	case subscribeErrMsg:
		m.runtimeStatus = runtime.RuntimeCrashed
		m.markDirty()
		return m, nil

	case eventStreamClosedMsg:
		if m.runtimeStatus != runtime.RuntimeCrashed {
			m.runtimeStatus = runtime.RuntimeStopped
			m.markDirty()
		}
		return m, nil

	case tickMsg:
		if m.flushStreaming() {
			m.markDirty()
		}
		return m, TickCmd()

	case tea.KeyMsg:
		m.markDirty()
		return m, m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.markDirty()
		return m, nil
	}

	return m, nil
}

// handleKey routes keypresses. Quit always works; otherwise the approval modal
// and session panel take priority over chat keys so typing/Enter/shortcuts
// cannot leak through an open modal.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if key.Matches(msg, m.keys.Quit) {
		return tea.Quit
	}
	if m.permission.Visible() {
		return m.handleApprovalKey(msg)
	}
	if m.sessionPanel.Visible {
		return m.handleSessionKey(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Submit):
		return m.submit()
	case key.Matches(msg, m.keys.Newline):
		m.input += "\n"
		return nil
	case key.Matches(msg, m.keys.Sessions):
		return m.toggleSessions()
	case key.Matches(msg, m.keys.ClearScreen):
		m.clearChat()
		return nil
	case key.Matches(msg, m.keys.ToggleThink):
		m.reasoningExpanded = !m.reasoningExpanded
		return nil
	default:
		m.handleTyping(msg)
		return nil
	}
}

// handleApprovalKey routes keys while the approval modal is open. Only
// Allow-once / Always / Reject are consumed; every other key is swallowed.
// While a reply is already in flight, all keys are swallowed so the same
// approval cannot be replied to twice.
func (m *Model) handleApprovalKey(msg tea.KeyMsg) tea.Cmd {
	if m.approvalPending {
		return nil
	}
	switch {
	case key.Matches(msg, m.keys.AllowOnce):
		return m.replyApproval(runtime.ApprovalOnce)
	case key.Matches(msg, m.keys.AllowAlways):
		return m.replyApproval(runtime.ApprovalAlways)
	case key.Matches(msg, m.keys.Reject), key.Matches(msg, m.keys.Esc):
		return m.replyApproval(runtime.ApprovalReject)
	}
	return nil
}

// replyApproval maps a decision to a Codea ApprovalReply and issues the async
// ReplyApproval command. The UI never fabricates a vendor "remember" flag;
// once/always/reject are the only decisions exposed here.
func (m *Model) replyApproval(decision runtime.ApprovalDecision) tea.Cmd {
	if m.permission.Request == nil {
		return nil
	}
	m.approvalPending = true
	return ReplyApprovalCmd(m.runtimeClient, runtime.ApprovalID(m.permission.Request.ID), runtime.ApprovalReply{Decision: decision})
}

// handleSessionKey routes keys while the session panel is open.
func (m *Model) handleSessionKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.sessionPanel.MoveUp()
		m.sessionNotice = ""
		return nil
	case key.Matches(msg, m.keys.Down):
		m.sessionPanel.MoveDown()
		m.sessionNotice = ""
		return nil
	case key.Matches(msg, m.keys.Submit):
		return m.resumeSelectedSession()
	case key.Matches(msg, m.keys.Esc), key.Matches(msg, m.keys.Sessions):
		m.sessionPanel.Close()
		m.sessionNotice = ""
		return nil
	}
	return nil
}

// toggleSessions opens the session panel (fetching sessions) or closes it.
func (m *Model) toggleSessions() tea.Cmd {
	if m.sessionPanel.Visible {
		m.sessionPanel.Close()
		m.sessionNotice = ""
		return nil
	}
	m.sessionPanel.Visible = true
	m.sessionPanel.Items = nil
	m.sessionPanel.Cursor = 0
	m.sessionNotice = ""
	return ListSessionsCmd(m.runtimeClient)
}

// resumeSelectedSession resumes the session under the cursor. It is blocked
// while a response is still streaming. On selection it kicks off an async
// history load (LoadSessionHistoryCmd); the actual switch happens in the
// loadHistoryResultMsg handler once history arrives.
func (m *Model) resumeSelectedSession() tea.Cmd {
	item, ok := m.sessionPanel.Selected()
	if !ok || item.Active {
		m.sessionPanel.Close()
		m.sessionNotice = ""
		return nil
	}
	if m.isStreaming {
		m.sessionNotice = "Finish or cancel the current response before switching sessions."
		return nil
	}
	m.pendingResumeID = item.ID
	m.sessionNotice = ""
	return LoadSessionHistoryCmd(m.runtimeClient, item.ID)
}

// resumeSession switches the current session, resets all transient UI state so
// the previous session's in-flight streaming, reasoning, tools, and pending
// prompt cannot leak into the resumed session, and rehydrates the chat view
// from the loaded history.
func (m *Model) resumeSession(id runtime.SessionID, history []runtime.Message) {
	m.sessionID = id
	m.isStreaming = false
	m.pendingPrompt = nil
	m.streamBuf.Reset()
	m.reasoningBuf.Reset()
	m.reasoningActive = false
	m.reasoningContent = ""
	m.reasoningDuration = 0
	m.reasoningExpanded = false
	m.tools = make([]ToolActivity, 0)
	m.proc.Reset()
	m.messages = historyToChatMessages(history)
	m.sessionNotice = ""
}

// historyToChatMessages maps Codea-domain session messages into chat messages
// for rehydration. Historical turns are complete, so every message is marked
// finished; an unknown role falls back to informational display.
func historyToChatMessages(history []runtime.Message) []ChatMessage {
	out := make([]ChatMessage, 0, len(history))
	for _, msg := range history {
		out = append(out, ChatMessage{
			Role:     messageRole(msg.Role),
			Content:  msg.Content,
			Finished: true,
		})
	}
	return out
}

// messageRole maps a message role string to a Codea Role.
func messageRole(role string) Role {
	switch role {
	case "user":
		return RoleUser
	case "assistant":
		return RoleAssistant
	default:
		return RoleInfo
	}
}

// sessionItems converts Codea-domain sessions into session list items.
func sessionItems(sessions []runtime.Session) []components.SessionItem {
	items := make([]components.SessionItem, len(sessions))
	for i, s := range sessions {
		items[i] = components.SessionItem{
			ID:        runtime.SessionID(s.ID),
			Title:     s.Title,
			UpdatedAt: s.UpdatedAt,
		}
	}
	return items
}

// handleTyping appends printable runes and handles backspace.
func (m *Model) handleTyping(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyBackspace:
		m.input = deleteLastRune(m.input)
	case tea.KeyRunes:
		m.input += string(msg.Runes)
	case tea.KeySpace:
		m.input += " "
	}
}

// submit sends the current input as a prompt. An empty (whitespace-only) input
// is ignored. A new session is created on the first submit; subsequent submits
// reuse it.
func (m *Model) submit() tea.Cmd {
	if m.isStreaming {
		return nil
	}
	if strings.TrimSpace(m.input) == "" {
		return nil
	}
	m.messages = append(m.messages,
		ChatMessage{Role: RoleUser, Content: m.input, Finished: true},
		ChatMessage{Role: RoleAssistant},
	)
	m.isStreaming = true
	m.proc.Reset()
	m.reasoningActive = false
	m.reasoningContent = ""
	m.reasoningDuration = 0
	m.reasoningExpanded = false
	m.streamBuf.Reset()
	m.reasoningBuf.Reset()
	m.tools = make([]ToolActivity, 0)

	raw := m.input
	req := runtime.PromptRequest{
		MessageID: fmt.Sprintf("msg-%d", m.msgCounter),
		Agent:     "general",
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: raw}},
	}
	m.msgCounter++
	m.input = ""

	if m.sessionID == "" {
		m.pendingPrompt = &req
		return CreateSessionCmd(m.runtimeClient, strings.TrimSpace(raw))
	}
	return PromptCmd(m.runtimeClient, m.sessionID, req)
}

// processRuntimeEvent consumes one runtime event, updating streaming answer and
// reasoning state via the reasoning processor and tracking stream completion.
// Answer and reasoning deltas are buffered (not committed to the visible
// message/reasoning state) so a high-frequency token burst does not trigger a
// render per token; the returned bool reports whether any visible state changed.
func (m *Model) processRuntimeEvent(ev runtime.Event) bool {
	if !m.acceptsEvent(ev) {
		return false
	}
	dirty := false
	switch ev.Type {
	case eventTypeStepFinished:
		dirty = m.applyReasoningEvents(m.proc.Flush()) || dirty
		m.finishStreaming()
		dirty = true
	case eventTypeSessionError, eventTypeRuntimeError:
		dirty = m.applyReasoningEvents(m.proc.Process(ev)) || dirty
		m.finishStreaming()
		dirty = true
	case eventTypeToolCalled:
		m.addTool(ev)
		dirty = true
	case eventTypeToolSuccess:
		m.updateTool(ev, ToolSuccess)
		dirty = true
	case eventTypeToolFailed:
		m.updateTool(ev, ToolFailed)
		dirty = true
	case eventTypeApprovalRequested:
		if ev.Approval != nil {
			m.permission = components.NewPermissionModel(ev.Approval)
			m.approvalErr = ""
			m.approvalPending = false
			dirty = true
		}
	default:
		dirty = m.applyReasoningEvents(m.proc.Process(ev)) || dirty
	}
	return dirty
}

// acceptsEvent reports whether a runtime event should be consumed. Session-scoped
// events (non-empty SessionID) are consumed only when they belong to the current
// session; events with an empty SessionID are genuine global/runtime events and
// are always accepted.
func (m *Model) acceptsEvent(ev runtime.Event) bool {
	if ev.SessionID == "" {
		return true
	}
	return ev.SessionID == string(m.sessionID)
}

// applyReasoningEvents folds processor output into the visible answer/reasoning
// state. Answer and reasoning deltas are buffered (not committed) so a token
// burst does not trigger one render per token; it returns whether any visible
// state changed.
func (m *Model) applyReasoningEvents(events []reasoning.Event) bool {
	dirty := false
	for _, pe := range events {
		switch pe.Kind {
		case reasoning.EventAnswerDelta:
			m.streamBuf.WriteString(pe.Content)
		case reasoning.EventReasoningStart:
			m.reasoningActive = true
			m.reasoningContent = ""
			m.reasoningBuf.Reset()
			dirty = true
		case reasoning.EventReasoningDelta:
			m.reasoningBuf.WriteString(pe.Content)
		case reasoning.EventReasoningEnd:
			m.reasoningActive = false
			m.reasoningDuration = pe.Duration
			m.reasoningExpanded = false
			dirty = true
		}
	}
	return dirty
}

// flushStreaming commits buffered answer and reasoning deltas into the visible
// state. It reports whether anything was flushed so callers only invalidate the
// render cache when the view actually changed (idle ticks do not re-render).
func (m *Model) flushStreaming() bool {
	dirty := false
	if m.streamBuf.Len() > 0 {
		m.appendAnswer(m.streamBuf.String())
		m.streamBuf.Reset()
		dirty = true
	}
	if m.reasoningBuf.Len() > 0 {
		m.reasoningContent += m.reasoningBuf.String()
		m.reasoningBuf.Reset()
		dirty = true
	}
	return dirty
}

// appendAnswer appends a streaming chunk to the single in-flight assistant
// message, never creating a new message per delta.
func (m *Model) appendAnswer(content string) {
	if content == "" {
		return
	}
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == RoleAssistant && !m.messages[i].Finished {
			m.messages[i].Content += content
			return
		}
	}
	m.messages = append(m.messages, ChatMessage{Role: RoleAssistant, Content: content})
}

// finishStreaming flushes any buffered streaming content, then marks the
// in-flight assistant message finished and clears the streaming flag.
func (m *Model) finishStreaming() {
	m.flushStreaming()
	m.isStreaming = false
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == RoleAssistant && !m.messages[i].Finished {
			m.messages[i].Finished = true
			return
		}
	}
}

// deleteLastRune removes the final rune from s, handling multi-byte UTF-8.
func deleteLastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

// addTool records a newly started tool invocation. The real OpenCode tool
// lifecycle emits multiple tool.called events for one call (pending → running,
// sharing one callID), so this upserts by callID to keep a single ToolActivity
// per invocation; appending a duplicate would leave a second entry stuck in
// running when the terminal event updates only the first match.
func (m *Model) addTool(ev runtime.Event) {
	if ev.Tool == nil {
		return
	}
	for i := range m.tools {
		if m.tools[i].CallID == ev.Tool.CallID {
			if ev.Tool.Name != "" {
				m.tools[i].Name = ev.Tool.Name
			}
			m.tools[i].Status = ToolRunning
			return
		}
	}
	m.tools = append(m.tools, ToolActivity{Name: ev.Tool.Name, CallID: ev.Tool.CallID, Status: ToolRunning})
}

// updateTool marks a previously started tool invocation as completed.
func (m *Model) updateTool(ev runtime.Event, status ToolStatus) {
	if ev.Tool == nil {
		return
	}
	for i := range m.tools {
		if m.tools[i].CallID == ev.Tool.CallID {
			m.tools[i].Status = status
			return
		}
	}
}

// clearChat resets the visible conversation, tools, and reasoning state.
func (m *Model) clearChat() {
	m.messages = make([]ChatMessage, 0)
	m.tools = make([]ToolActivity, 0)
	m.reasoningActive = false
	m.reasoningContent = ""
	m.reasoningExpanded = false
	m.reasoningDuration = 0
	m.streamBuf.Reset()
	m.reasoningBuf.Reset()
}
