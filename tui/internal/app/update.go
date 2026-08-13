package app

import (
	"fmt"
	"strings"

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
		return m, waitForEvent(msg.ch)

	case runtimeEventMsg:
		m.processRuntimeEvent(msg.ev)
		if m.eventCh != nil {
			return m, waitForEvent(m.eventCh)
		}
		return m, nil

	case promptResultMsg:
		if msg.err != nil {
			m.finishStreaming()
			return m, nil
		}
		if msg.sessionID != "" {
			m.sessionID = msg.sessionID
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

	case tea.KeyMsg:
		return m, m.handleKey(msg)
	}

	return m, nil
}

// handleKey routes keypresses to input handling or submission.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Submit):
		return m.submit()
	case key.Matches(msg, m.keys.Newline):
		m.input += "\n"
		return nil
	default:
		m.handleTyping(msg)
		return nil
	}
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
	if strings.TrimSpace(m.input) == "" {
		return nil
	}
	m.messages = append(m.messages,
		ChatMessage{Role: RoleUser, Content: m.input, Finished: true},
		ChatMessage{Role: RoleAssistant},
	)
	m.isStreaming = true

	raw := m.input
	req := runtime.PromptRequest{
		MessageID: fmt.Sprintf("msg-%d", m.msgCounter),
		Agent:     "general",
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: raw}},
	}
	m.msgCounter++
	m.input = ""

	if m.sessionID == "" {
		return CreateSessionAndPromptCmd(m.runtimeClient, strings.TrimSpace(raw), req)
	}
	return PromptCmd(m.runtimeClient, m.sessionID, req)
}

// processRuntimeEvent consumes one runtime event, updating streaming answer and
// reasoning state via the reasoning processor and tracking stream completion.
func (m *Model) processRuntimeEvent(ev runtime.Event) {
	switch ev.Type {
	case eventTypeStepFinished, eventTypeSessionError, eventTypeRuntimeError:
		m.finishStreaming()
	}

	for _, pe := range m.proc.Process(ev) {
		switch pe.Kind {
		case reasoning.EventAnswerDelta:
			m.appendAnswer(pe.Content)
		case reasoning.EventReasoningStart:
			m.reasoningActive = true
			m.reasoningContent = ""
		case reasoning.EventReasoningDelta:
			m.reasoningContent += pe.Content
		case reasoning.EventReasoningEnd:
			m.reasoningActive = false
			m.reasoningDuration = pe.Duration
			m.reasoningExpanded = false
		}
	}
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

// finishStreaming marks the in-flight assistant message finished and clears the
// streaming flag.
func (m *Model) finishStreaming() {
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
