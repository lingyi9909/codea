package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

const internalModelCheckTitlePrefix = "__codea_internal_model_check__"

const fixedQualificationInstruction = "Run the four Codea model qualification probes in this exact order: probe_tool_call, probe_structured, probe_patch, probe_plan. Use only fixed public probe inputs. Each probe may be attempted at most twice, retrying only after a failed first attempt. Do not read project files, execute commands, call non-probe tools, or include chain-of-thought."

type ProbeOutcome struct {
	Attempts int
	Passed   bool
}

type ModelCheckState struct {
	Active      bool
	Terminal    bool
	SessionID   runtime.SessionID
	Model       runtime.ModelRef
	ToolCalling ProbeOutcome
	Structured  ProbeOutcome
	Patch       ProbeOutcome
	Planning    ProbeOutcome
}

type modelCheckModelsMsg struct {
	models []runtime.Model
	err    error
}

type modelCheckSessionCreatedMsg struct {
	sessionID runtime.SessionID
	err       error
}

type modelCheckPromptResultMsg struct {
	sessionID runtime.SessionID
	err       error
}

type modelCheckProfileSavedMsg struct {
	profile modelprofile.ModelCapabilityProfile
	err     error
}

func (m *Model) SetModelProfileStore(store *modelprofile.Store) {
	m.modelProfileStore = store
}

func (m *Model) beginModelCheck(arguments string) tea.Cmd {
	if strings.TrimSpace(arguments) != "" {
		m.appendInfo("Usage: /model-check")
		return nil
	}
	if m.isStreaming {
		m.appendInfo("Finish or cancel the current response before running /model-check.")
		return nil
	}
	if m.permission.Visible() || m.approvalPending {
		m.appendInfo("Resolve the active approval before running /model-check.")
		return nil
	}
	if m.modelCheck.Active {
		m.appendInfo("Model qualification is already in progress.")
		return nil
	}
	if m.modelProfileStore == nil {
		m.appendInfo("Model qualification unavailable: model profile store is not configured.")
		return nil
	}

	m.modelCheck = ModelCheckState{Active: true}
	if ref, ok := m.sessionModels[m.sessionID]; ok && ref.ProviderID != "" && ref.ModelID != "" {
		m.modelCheck.Model = ref
		return createModelCheckSessionCmd(m.runtimeClient, ref)
	}
	return listModelCheckModelsCmd(m.runtimeClient)
}

func listModelCheckModelsCmd(client runtime.AgentRuntime) tea.Cmd {
	return func() tea.Msg {
		models, err := client.ListModels(context.Background())
		return modelCheckModelsMsg{models: models, err: err}
	}
}

func createModelCheckSessionCmd(client runtime.AgentRuntime, ref runtime.ModelRef) tea.Cmd {
	return func() tea.Msg {
		title := internalModelCheckTitlePrefix + ref.ProviderID + "/" + ref.ModelID
		session, err := client.CreateSession(context.Background(), runtime.CreateSessionRequest{Title: title})
		if err != nil {
			return modelCheckSessionCreatedMsg{err: err}
		}
		return modelCheckSessionCreatedMsg{sessionID: runtime.SessionID(session.ID)}
	}
}

func modelCheckPrompt(ref runtime.ModelRef) runtime.PromptRequest {
	return runtime.PromptRequest{
		MessageID: "__codea_model_check__",
		Agent:     "model-evaluator",
		Model:     &ref,
		Parts: []runtime.PromptPart{runtime.TextPart{
			Synthetic: true,
			Metadata:  map[string]any{"codea.kind": "model-check"},
			Text:      fixedQualificationInstruction,
		}},
	}
}

func promptModelCheckCmd(client runtime.AgentRuntime, sessionID runtime.SessionID, ref runtime.ModelRef) tea.Cmd {
	return func() tea.Msg {
		err := client.Prompt(context.Background(), sessionID, modelCheckPrompt(ref))
		return modelCheckPromptResultMsg{sessionID: sessionID, err: err}
	}
}

func saveModelCheckProfileCmd(store *modelprofile.Store, profile modelprofile.ModelCapabilityProfile) tea.Cmd {
	return func() tea.Msg {
		return modelCheckProfileSavedMsg{profile: profile, err: store.Save(profile)}
	}
}

func uniqueDefaultModel(models []runtime.Model) (runtime.ModelRef, bool) {
	var ref runtime.ModelRef
	count := 0
	for _, model := range models {
		if model.Default && model.Ref.ProviderID != "" && model.Ref.ModelID != "" {
			ref = model.Ref
			count++
		}
	}
	return ref, count == 1
}

func (m *Model) handleModelCheckMessage(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case modelCheckModelsMsg:
		if !m.modelCheck.Active || m.modelCheck.SessionID != "" {
			return true, nil
		}
		if msg.err != nil {
			m.failModelCheck("Failed to resolve Runtime models: " + msg.err.Error())
			return true, nil
		}
		// A non-nil slice records that Runtime model discovery completed even
		// when the Runtime returned zero models. Normal first-task resolution
		// can therefore distinguish unresolved from resolved-empty state.
		m.runtimeModels = append([]runtime.Model{}, msg.models...)
		ref, ok := uniqueDefaultModel(msg.models)
		if !ok {
			m.failModelCheck("Model qualification requires an exact model. Use /model first, or configure exactly one Runtime default model.")
			return true, nil
		}
		m.modelCheck.Model = ref
		return true, createModelCheckSessionCmd(m.runtimeClient, ref)

	case modelCheckSessionCreatedMsg:
		if !m.modelCheck.Active {
			return true, nil
		}
		if msg.err != nil {
			m.failModelCheck("Model qualification session creation failed: " + msg.err.Error())
			return true, nil
		}
		m.modelCheck.SessionID = msg.sessionID
		return true, promptModelCheckCmd(m.runtimeClient, msg.sessionID, m.modelCheck.Model)

	case modelCheckPromptResultMsg:
		if !m.modelCheck.Active || msg.sessionID != m.modelCheck.SessionID {
			return true, nil
		}
		if msg.err != nil {
			m.failModelCheck("Model qualification failed before a bounded result: " + msg.err.Error())
		}
		return true, nil

	case modelCheckProfileSavedMsg:
		if !m.modelCheck.Active || !m.modelCheck.Terminal || msg.profile.Model != m.modelCheck.Model {
			return true, nil
		}
		if msg.err != nil {
			m.failModelCheck("Model qualification completed but profile persistence failed: " + msg.err.Error())
			return true, nil
		}
		m.modelCheck.Active = false
		m.appendInfo(formatModelCheckResult(msg.profile))
		return true, nil
	}
	return false, nil
}

func (m *Model) handleModelCheckEvent(ev runtime.Event) (bool, tea.Cmd) {
	if !m.modelCheck.Active || m.modelCheck.SessionID == "" || runtime.SessionID(ev.SessionID) != m.modelCheck.SessionID {
		return false, nil
	}
	// Once a bounded qualification result has been reached, all later events
	// from the internal session are consumed while persistence/cancellation
	// finishes. In particular, the cancellation we issue after two failures
	// must not turn an already-determined WEAK result into an aborted result.
	if m.modelCheck.Terminal {
		return true, nil
	}
	if ev.Type == eventTypeRuntimeError || ev.Type == eventTypeSessionError {
		m.failModelCheck("Model qualification was interrupted by an internal Runtime/session error.")
		return true, nil
	}

	if terminalWeak := m.recordProbeTerminalEvent(ev); terminalWeak {
		return m.finishModelCheckEarlyWeak()
	}
	if ev.Type != eventTypeStepFinished {
		return true, nil
	}
	return m.finishModelCheck(false)
}

// recordProbeTerminalEvent derives the mechanical attempt count from Runtime
// terminal tool events, not from the plugin's execute-local counter. OpenCode
// v1.18.11 performs registered-tool schema validation before plugin execute;
// a schema rejection therefore arrives as tool.failed and still consumes one
// qualification attempt. Successful evidence must additionally carry the
// plugin's canonical codeaProbe metadata for the exact registered tool ID.
func (m *Model) recordProbeTerminalEvent(ev runtime.Event) bool {
	if ev.Tool == nil || (ev.Type != eventTypeToolSuccess && ev.Type != eventTypeToolFailed) {
		return false
	}
	outcome, expectedProbe := m.probeOutcomeForToolID(strings.TrimSpace(ev.Tool.Name))
	if outcome == nil {
		return false
	}
	if outcome.Attempts >= 2 {
		// A third terminal invocation is a protocol violation. Fail closed and
		// terminate the internal qualification so retries cannot continue.
		outcome.Passed = false
		return true
	}
	outcome.Attempts++
	if ev.Type == eventTypeToolSuccess {
		metadata := ev.Tool.Metadata
		attempt := strings.TrimSpace(metadata["codeaProbeAttempt"])
		if strings.TrimSpace(metadata["codeaProbe"]) == expectedProbe &&
			strings.TrimSpace(metadata["codeaProbeResult"]) == "pass" &&
			(attempt == "1" || attempt == "2") {
			outcome.Passed = true
		}
	}
	return outcome.Attempts >= 2 && !outcome.Passed
}

func (m *Model) probeOutcomeForToolID(toolID string) (*ProbeOutcome, string) {
	switch toolID {
	case "probe_tool_call":
		return &m.modelCheck.ToolCalling, "tool_call"
	case "probe_structured":
		return &m.modelCheck.Structured, "structured"
	case "probe_patch":
		return &m.modelCheck.Patch, "patch"
	case "probe_plan":
		return &m.modelCheck.Planning, "planning"
	default:
		return nil, ""
	}
}

func (m *Model) finishModelCheckEarlyWeak() (bool, tea.Cmd) {
	return m.finishModelCheck(true)
}

func (m *Model) finishModelCheck(cancelInternal bool) (bool, tea.Cmd) {
	m.modelCheck.Terminal = true
	profile := m.completedModelProfile(time.Now().UTC())
	if m.modelProfileStore == nil {
		m.failModelCheck("Model qualification completed but profile store is unavailable.")
		return true, nil
	}
	save := saveModelCheckProfileCmd(m.modelProfileStore, profile)
	if !cancelInternal || m.runtimeClient == nil {
		return true, save
	}
	return true, tea.Batch(CancelResponseCmd(m.runtimeClient, m.modelCheck.SessionID), save)
}

func levelForProbe(outcome ProbeOutcome) modelprofile.CapabilityLevel {
	if outcome.Passed && outcome.Attempts <= 1 { return modelprofile.CapabilityStrong }
	if outcome.Passed && outcome.Attempts == 2 { return modelprofile.CapabilityMedium }
	return modelprofile.CapabilityWeak
}

func (m *Model) completedModelProfile(now time.Time) modelprofile.ModelCapabilityProfile {
	tool := levelForProbe(m.modelCheck.ToolCalling)
	structured := levelForProbe(m.modelCheck.Structured)
	patch := levelForProbe(m.modelCheck.Patch)
	planning := levelForProbe(m.modelCheck.Planning)
	return modelprofile.ModelCapabilityProfile{
		Model: m.modelCheck.Model,
		ToolCalling: tool,
		StructuredOutput: structured,
		PatchFollowing: patch,
		Planning: planning,
		Overall: modelprofile.Aggregate(tool, structured, patch, planning),
		QualifiedAt: now,
		Version: modelprofile.ProfileVersion,
	}
}

func formatModelCheckResult(profile modelprofile.ModelCapabilityProfile) string {
	strategy := modelprofile.StrategyForProfile(&profile)
	return fmt.Sprintf("Model capability: %s\nTool calling: %s\nStructured output: %s\nPatch following: %s\nPlanning: %s\nStrategy: %s", strings.ToUpper(string(profile.Overall)), strings.ToUpper(string(profile.ToolCalling)), strings.ToUpper(string(profile.StructuredOutput)), strings.ToUpper(string(profile.PatchFollowing)), strings.ToUpper(string(profile.Planning)), strategy.Level)
}

func (m *Model) failModelCheck(message string) {
	m.modelCheck.Active = false
	m.modelCheck.Terminal = false
	m.appendInfo(message + " Medium fallback remains active; no completed profile was persisted.")
}

func filterInternalSessions(sessions []runtime.Session) []runtime.Session {
	filtered := make([]runtime.Session, 0, len(sessions))
	for _, session := range sessions {
		if strings.HasPrefix(session.Title, internalModelCheckTitlePrefix) {
			continue
		}
		filtered = append(filtered, session)
	}
	return filtered
}
