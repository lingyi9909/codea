package app

import (
	"context"
	"fmt"
	"strings"

	"codea/tui/internal/doctor"
	"codea/tui/internal/runtime"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// WorkspaceInfo is safe composition-root metadata shown by /status.
// It must never contain credentials or raw provider configuration.
type WorkspaceInfo struct {
	CodeaVersion    string
	RuntimeProvider string
	Project         string
	SkillMode       string
}

type modelPickerModel struct {
	Visible   bool
	Items     []runtime.Model
	Cursor    int
	SessionID runtime.SessionID
}

func (p *modelPickerModel) Close() {
	p.Visible = false
	p.Items = nil
	p.Cursor = 0
	p.SessionID = ""
}

func (p *modelPickerModel) MoveUp() {
	if p.Cursor > 0 {
		p.Cursor--
	}
}

func (p *modelPickerModel) MoveDown() {
	if p.Cursor+1 < len(p.Items) {
		p.Cursor++
	}
}

func (p *modelPickerModel) Selected() (runtime.Model, bool) {
	if !p.Visible || p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return runtime.Model{}, false
	}
	return p.Items[p.Cursor], true
}

func (m *Model) SetWorkspaceInfo(info WorkspaceInfo) {
	m.workspaceInfo = info
	m.markDirty()
}

func (m *Model) SetDoctorService(service *doctor.Service) {
	m.doctorService = service
}

type listModelsResultMsg struct {
	sessionID runtime.SessionID
	models    []runtime.Model
	err       error
}

type compactSessionResultMsg struct {
	sessionID runtime.SessionID
	err       error
}

type workspaceStatusResultMsg struct {
	health runtime.HealthInfo
	err    error
}

type doctorServiceResultMsg struct {
	report doctor.Report
}

func ListModelsCmd(client runtime.AgentRuntime, sessionID runtime.SessionID) tea.Cmd {
	return func() tea.Msg {
		models, err := client.ListModels(context.Background())
		return listModelsResultMsg{sessionID: sessionID, models: models, err: err}
	}
}

func CompactSessionCmd(client runtime.AgentRuntime, sessionID runtime.SessionID) tea.Cmd {
	return func() tea.Msg {
		return compactSessionResultMsg{sessionID: sessionID, err: client.CompactSession(context.Background(), sessionID)}
	}
}

func RuntimeWorkspaceStatusCmd(client runtime.AgentRuntime) tea.Cmd {
	return func() tea.Msg {
		health, err := client.Health(context.Background())
		return workspaceStatusResultMsg{health: health, err: err}
	}
}

func DoctorServiceCmd(service *doctor.Service) tea.Cmd {
	return func() tea.Msg {
		return doctorServiceResultMsg{report: service.Run(context.Background())}
	}
}

// handleRuntimeWorkspaceMessage centralizes Codea-owned asynchronous workspace
// results, including Task 28 Repo Context and Task 31 checkpoint preparation,
// before the main Update switch handles Runtime lifecycle messages.
func (m *Model) handleRuntimeWorkspaceMessage(msg tea.Msg) (bool, tea.Cmd) {
	// A tick still falls through to the main Update switch so streaming buffers
	// are flushed and the next TickCmd is scheduled. We only advance the visual
	// spinner here. During approval waiting the spinner deliberately stops.
	if _, ok := msg.(tickMsg); ok && m.isStreaming && !m.approvalWaiting() {
		m.spinnerFrame = (m.spinnerFrame + 1) % len(workingSpinnerFrames)
		m.markDirty()
	}

	if handled, cmd := m.handleCheckpointMessage(msg); handled {
		return true, cmd
	}
	if handled, cmd := m.handleProfessionalWorkspaceMessage(msg); handled {
		return true, cmd
	}

	switch msg := msg.(type) {
	case repoContextResultMsg:
		return true, m.handleRepoContextResult(msg)

	case listModelsResultMsg:
		if msg.sessionID == "" || msg.sessionID != m.sessionID {
			return true, nil
		}
		if msg.err != nil {
			m.appendInfo("Failed to load runtime models: " + msg.err.Error())
			return true, nil
		}
		m.modelPicker = modelPickerModel{Visible: true, Items: append([]runtime.Model(nil), msg.models...), SessionID: msg.sessionID}
		if len(msg.models) == 0 {
			m.modelPicker.Close()
			m.appendInfo("No runtime models are available.")
		}
		m.markDirty()
		return true, nil

	case compactSessionResultMsg:
		if msg.sessionID != m.sessionID {
			return true, nil
		}
		if msg.err != nil {
			m.appendInfo("Context compaction failed: " + msg.err.Error())
			return true, nil
		}
		m.appendInfo("Context compacted for the current session.")
		return true, nil

	case workspaceStatusResultMsg:
		m.appendInfo(m.runtimeWorkspaceStatus(msg.health, msg.err))
		return true, nil

	case doctorServiceResultMsg:
		m.appendInfo(strings.TrimSpace(doctor.FormatText(msg.report)))
		return true, nil
	}
	return false, nil
}

func (m *Model) handleModelPickerKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.modelPicker.MoveUp()
	case key.Matches(msg, m.keys.Down):
		m.modelPicker.MoveDown()
	case key.Matches(msg, m.keys.Esc):
		m.modelPicker.Close()
	case key.Matches(msg, m.keys.Submit):
		selected, ok := m.modelPicker.Selected()
		sessionID := m.modelPicker.SessionID
		m.modelPicker.Close()
		if !ok || sessionID == "" || sessionID != m.sessionID {
			m.appendInfo("Model selection expired because the active session changed.")
			return nil
		}
		m.sessionModels[sessionID] = selected.Ref
		m.appendInfo(fmt.Sprintf("Model selected for this session: %s/%s", selected.Ref.ProviderID, selected.Ref.ModelID))
	}
	m.markDirty()
	return nil
}

func (m *Model) runtimeWorkspaceStatus(health runtime.HealthInfo, healthErr error) string {
	provider := strings.TrimSpace(m.workspaceInfo.RuntimeProvider)
	if provider == "" {
		provider = "runtime"
	}
	version := health.Version
	healthText := "healthy"
	if healthErr != nil {
		healthText = "unavailable: " + healthErr.Error()
		version = "unknown"
	} else if !health.Healthy {
		healthText = "unhealthy"
	}
	session := "none"
	if m.sessionID != "" {
		session = string(m.sessionID)
	}
	model := "runtime-default"
	if ref, ok := m.sessionModels[m.sessionID]; ok {
		model = ref.ProviderID + "/" + ref.ModelID
	}
	agent := strings.TrimSpace(m.currentAgent)
	if agent == "" {
		agent = "general"
	}
	skills := "none"
	if len(m.loadedSkillIDs) > 0 {
		skills = strings.Join(m.loadedSkillIDs, ", ")
	}
	caps := m.runtimeClient.Capabilities()
	return fmt.Sprintf(
		"Codea: %s\nRuntime: %s %s (%s)\nProject: %s\nSession: %s\nAgent: %s\nModel: %s\nSkill Mode: %s\nLoaded Skills: %s\nCapabilities:\n- Streaming: %t\n- Reasoning: %t\n- Tool approval: %t\n- Compaction: %t",
		m.workspaceInfo.CodeaVersion,
		provider,
		version,
		healthText,
		m.workspaceInfo.Project,
		session,
		agent,
		model,
		m.workspaceInfo.SkillMode,
		skills,
		caps.Streaming,
		caps.Reasoning,
		caps.ToolApproval,
		caps.ContextCompaction,
	)
}

func (m *Model) modelPickerView() string {
	if !m.modelPicker.Visible {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nModels (current session)\n")
	for i, model := range m.modelPicker.Items {
		prefix := "  "
		if i == m.modelPicker.Cursor {
			prefix = "> "
		}
		defaultLabel := ""
		if model.Default {
			defaultLabel = " [runtime default]"
		}
		fmt.Fprintf(&b, "%s%s/%s — %s%s\n", prefix, model.Ref.ProviderID, model.Ref.ModelID, model.Name, defaultLabel)
	}
	b.WriteString("↑/↓ select · Enter apply · Esc close")
	return b.String()
}
