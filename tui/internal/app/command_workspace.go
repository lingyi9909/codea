package app

import (
	"fmt"
	"strings"

	"codea/tui/internal/command"
	"codea/tui/internal/runtime"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type commandPaletteModel struct {
	Visible bool
	Items   []command.Definition
	Cursor  int
}

func (p *commandPaletteModel) Close() {
	p.Visible = false
	p.Items = nil
	p.Cursor = 0
}

func (p *commandPaletteModel) MoveUp() {
	if len(p.Items) == 0 {
		return
	}
	if p.Cursor > 0 {
		p.Cursor--
	}
}

func (p *commandPaletteModel) MoveDown() {
	if len(p.Items) == 0 {
		return
	}
	if p.Cursor < len(p.Items)-1 {
		p.Cursor++
	}
}

func (p *commandPaletteModel) Selected() (command.Definition, bool) {
	if !p.Visible || p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return command.Definition{}, false
	}
	return p.Items[p.Cursor], true
}

func defaultCommandRegistry() *command.Registry {
	reg := command.NewRegistry()
	for _, def := range command.BuiltinCommands() {
		_ = reg.Register(def)
	}
	return reg
}

// SetCommandRegistry replaces the default built-in registry with the composed
// Built-in + Enterprise + Project registry created by cmd/codea.
func (m *Model) SetCommandRegistry(reg *command.Registry) {
	if reg == nil {
		return
	}
	m.commandRegistry = reg
	m.refreshCommandPalette()
	m.markDirty()
}

func (m *Model) refreshCommandPalette() {
	if m.commandRegistry == nil || !strings.HasPrefix(m.input, "/") {
		m.commandPalette.Close()
		return
	}
	m.commandPalette.Visible = true
	m.commandPalette.Items = m.commandRegistry.Filter(m.input)
	if len(m.commandPalette.Items) == 0 {
		m.commandPalette.Cursor = 0
		return
	}
	if m.commandPalette.Cursor >= len(m.commandPalette.Items) {
		m.commandPalette.Cursor = len(m.commandPalette.Items) - 1
	}
	if m.commandPalette.Cursor < 0 {
		m.commandPalette.Cursor = 0
	}
}

func (m *Model) handleCommandPaletteKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.commandPalette.MoveUp()
		return nil
	case key.Matches(msg, m.keys.Down):
		m.commandPalette.MoveDown()
		return nil
	case key.Matches(msg, m.keys.Esc):
		m.commandPalette.Close()
		return nil
	case key.Matches(msg, m.keys.Submit):
		if def, ok := m.commandPalette.Selected(); ok {
			m.input = selectCommandName(m.input, def.Name)
		}
		m.commandPalette.Close()
		return m.submit()
	}

	switch msg.Type {
	case tea.KeyBackspace, tea.KeyRunes, tea.KeySpace:
		m.handleTyping(msg)
	}
	return nil
}

func selectCommandName(input, name string) string {
	for i, ch := range input {
		if i == 0 {
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			return "/" + name + input[i:]
		}
	}
	return "/" + name
}

func (m *Model) submitCommand(raw string) tea.Cmd {
	m.commandPalette.Close()
	out, err := m.commandRegistry.Execute(raw)
	if err != nil {
		m.input = ""
		m.appendInfo("Command error: " + err.Error())
		return nil
	}

	switch out.Kind {
	case command.OutcomePrompt:
		if m.isStreaming {
			m.input = ""
			m.appendInfo("Finish or cancel the current response before running a prompt command.")
			return nil
		}
		agent := strings.TrimSpace(out.Agent)
		if agent == "" {
			agent = "general"
		}
		return m.startCommandPrompt(raw, out.Prompt, agent)
	case command.OutcomeAction:
		m.input = ""
		if out.Action == command.ActionView {
			return m.setViewMode(out.Arguments)
		}
		return m.executeWorkspaceAction(out.Action)
	default:
		m.input = ""
		m.appendInfo("Command error: unsupported command outcome")
		return nil
	}
}

func (m *Model) executeWorkspaceAction(action command.Action) tea.Cmd {
	switch action {
	case command.ActionHelp:
		m.appendInfo(m.commandHelp())
		return nil
	case command.ActionClear:
		m.clearChat()
		return nil
	case command.ActionStatus:
		return RuntimeWorkspaceStatusCmd(m.runtimeClient)
	case command.ActionSessions:
		return m.toggleSessions()
	case command.ActionSkills:
		return m.toggleSkills()
	case command.ActionAgents:
		if m.isStreaming {
			m.appendInfo("Finish or cancel the current response before changing agents.")
			return nil
		}
		return ListAgentsCmd(m.runtimeClient)
	case command.ActionModel:
		if m.sessionID == "" {
			m.appendInfo("No active session. Start a conversation before selecting a model.")
			return nil
		}
		if m.isStreaming {
			m.appendInfo("Finish or cancel the current response before changing models.")
			return nil
		}
		return ListModelsCmd(m.runtimeClient, m.sessionID)
	case command.ActionCompact:
		if m.sessionID == "" {
			m.appendInfo("No active session to compact.")
			return nil
		}
		if !m.runtimeClient.Capabilities().ContextCompaction {
			m.appendInfo("Context compaction is unsupported by the current Runtime.")
			return nil
		}
		if m.isStreaming {
			m.appendInfo("Finish or cancel the current response before compacting context.")
			return nil
		}
		return CompactSessionCmd(m.runtimeClient, m.sessionID)
	case command.ActionCancel:
		if !m.isStreaming || m.sessionID == "" {
			m.appendInfo("No active response to cancel.")
			return nil
		}
		return CancelResponseCmd(m.runtimeClient, m.sessionID)
	case command.ActionDoctor:
		if m.doctorService == nil {
			m.appendInfo("Codea Doctor is unavailable in this workspace.")
			return nil
		}
		return DoctorServiceCmd(m.doctorService)
	default:
		m.appendInfo("Command error: unsupported workspace action " + string(action))
		return nil
	}
}

func (m *Model) setViewMode(raw string) tea.Cmd {
	mode := ViewMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case ViewNormal, ViewVerbose, ViewFocus:
		m.viewMode = mode
		m.markDirty()
		return nil
	default:
		m.appendInfo("Usage: /view normal|verbose|focus")
		return nil
	}
}

func (m *Model) commandHelp() string {
	if m.commandRegistry == nil {
		return "No commands available."
	}
	defs := m.commandRegistry.Commands()
	lines := make([]string, 0, len(defs)+1)
	lines = append(lines, "Commands:")
	for _, def := range defs {
		description := strings.TrimSpace(def.Description)
		if description == "" {
			description = "custom prompt"
		}
		lines = append(lines, fmt.Sprintf("/%s — %s", def.Name, description))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) appendInfo(content string) {
	m.messages = append(m.messages, ChatMessage{Role: RoleInfo, Content: content, Finished: true})
	m.markDirty()
}

func (m *Model) activeAgent(requested string) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested
	}
	active := strings.TrimSpace(m.currentAgent)
	if active != "" {
		return active
	}
	return "general"
}

func (m *Model) startCommandPrompt(displayText, promptText, requestedAgent string) tea.Cmd {
	return m.startPromptWithAgent(displayText, promptText, m.activeAgent(requestedAgent))
}

// startPrompt is the ordinary natural-language path. Its third parameter is
// only a fallback for an empty persistent selection; currentAgent, chosen by
// /agents, remains authoritative for normal multi-turn conversation.
func (m *Model) startPrompt(displayText, promptText, fallbackAgent string) tea.Cmd {
	agent := strings.TrimSpace(m.currentAgent)
	if agent == "" {
		agent = strings.TrimSpace(fallbackAgent)
	}
	return m.startPromptWithAgent(displayText, promptText, m.activeAgent(agent))
}

func (m *Model) startPromptWithAgent(displayText, promptText, agent string) tea.Cmd {
	req := runtime.PromptRequest{
		MessageID: fmt.Sprintf("msg-%d", m.msgCounter),
		Agent:     agent,
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: promptText}},
	}
	modelLabel := ""
	if m.sessionID != "" {
		if selected, ok := m.sessionModels[m.sessionID]; ok {
			model := selected
			req.Model = &model
			modelLabel = strings.TrimSpace(selected.ModelID)
		}
	}

	// Bind visible turn identity to the actual PromptRequest. Professional
	// one-shot commands can therefore show code-reviewer without mutating the
	// persistent currentAgent selected through /agents.
	m.messages = append(m.messages,
		ChatMessage{Role: RoleUser, Content: displayText, Finished: true, TurnID: req.MessageID},
		ChatMessage{Role: RoleAssistant, TurnID: req.MessageID, Agent: req.Agent, Model: modelLabel},
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

	m.beginPromptTrace(req)
	m.msgCounter++
	m.input = ""
	m.startTaskMetric(req.Agent)

	if m.repoContextService != nil {
		return RepoContextCmd(m.repoContextService, repoPromptIntent{
			request:     req,
			displayText: displayText,
			promptText:  promptText,
			queryText:   promptText,
		})
	}

	if m.sessionID == "" {
		m.pendingPrompt = &req
		return CreateSessionCmd(m.runtimeClient, strings.TrimSpace(displayText))
	}
	return PromptCmd(m.runtimeClient, m.sessionID, req)
}
