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

const fixedQualificationInstruction = "Run the four Codea model qualification probes in this exact order: probe-state, probe-structured, probe-patch, probe-plan. Use only fixed public probe inputs. Each probe may be attempted at most twice, retrying only after a failed first attempt. Do not read project files, execute commands, call non-probe tools, or include chain-of-thought."

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
		m.runtimeModels = append([]runtime.Model(nil), msg.models...)
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
	if m.modelCheck.Terminal {
		return true, nil
	}
	if ev.Tool != nil {
		m.recordProbeEvent(ev.Tool.Metadata)
	}
	if ev.Type != runtime.EventType("step.finished") {
		return true, nil
	}
	m.modelCheck.Terminal = true
	profile := m.completedModelProfile(time.Now().UTC())
	if m.modelProfileStore == nil {
		m.failModelCheck("Model qualification completed but profile store is unavailable.")
		return true, nil
	}
	return true, saveModelCheckProfileCmd(m.modelProfileStore, profile)
}

func (m *Model) recordProbeEvent(metadata map[string]string) {
	probe := strings.TrimSpace(metadata["codeaProbe"])
	result := strings.TrimSpace(metadata["codeaProbeResult"])
	attemptText := strings.TrimSpace(metadata["codeaProbeAttempt"])
	attempt := 0
	if attemptText == "1" { attempt = 1 }
	if attemptText == "2" { attempt = 2 }
	if attempt == 0 || (result != "pass" && result != "fail") {
		return
	}
	var outcome *ProbeOutcome
	switch probe {
	case "state": outcome = &m.modelCheck.ToolCalling
	case "structured": outcome = &m.modelCheck.Structured
	case "patch": outcome = &m.modelCheck.Patch
	case "plan": outcome = &m.modelCheck.Planning
	default: return
	}
	if attempt > outcome.Attempts { outcome.Attempts = attempt }
	if result == "pass" { outcome.Passed = true }
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
