package app

import (
	"context"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func mediumModelStrategy() modelprofile.Strategy {
	return modelprofile.StrategyForLevel(modelprofile.CapabilityMedium)
}

// currentModelStrategy resolves only an exact provider+model identity. A
// display name is never sufficient to select a persisted profile.
func (m *Model) currentModelStrategy(req runtime.PromptRequest) modelprofile.Strategy {
	ref, ok := m.exactModelRefForStrategy(req)
	if !ok || m.modelProfileStore == nil {
		return mediumModelStrategy()
	}
	profile, found, err := m.modelProfileStore.Load(ref)
	if err != nil || !found {
		return mediumModelStrategy()
	}
	return modelprofile.StrategyForProfile(&profile)
}

func (m *Model) exactModelRefForStrategy(req runtime.PromptRequest) (runtime.ModelRef, bool) {
	if req.Model != nil && validModelRef(*req.Model) {
		return *req.Model, true
	}
	if ref, ok := m.sessionModels[m.sessionID]; ok && validModelRef(ref) {
		return ref, true
	}
	return uniqueDefaultModel(m.runtimeModels)
}

func validModelRef(ref runtime.ModelRef) bool {
	return ref.ProviderID != "" && ref.ModelID != ""
}

// runtimeModels == nil means Runtime defaults have not yet been resolved in
// this process. A non-nil empty slice means discovery completed but produced
// no uniquely usable default, so later prompts conservatively stay medium
// without repeatedly opening any picker or requiring /model. The lazy lookup
// is only needed when a profile store exists; legacy/no-profile paths retain
// their previous synchronous medium behavior.
func (m *Model) needsDefaultModelResolutionForPrompt() bool {
	if m.modelProfileStore == nil || m.runtimeClient == nil {
		return false
	}
	if ref, ok := m.sessionModels[m.sessionID]; ok && validModelRef(ref) {
		return false
	}
	return m.runtimeModels == nil
}

type defaultModelPromptModelsMsg struct {
	models      []runtime.Model
	err         error
	displayText string
	promptText  string
	agent       string
}

func resolveDefaultModelForPromptCmd(client runtime.AgentRuntime, displayText, promptText, agent string) tea.Cmd {
	return func() tea.Msg {
		models, err := client.ListModels(context.Background())
		return defaultModelPromptModelsMsg{
			models:      models,
			err:         err,
			displayText: displayText,
			promptText:  promptText,
			agent:       agent,
		}
	}
}

func modelStrategyPart(strategy modelprofile.Strategy) runtime.TextPart {
	if strategy.RepoMapMaxChars <= 0 {
		strategy = mediumModelStrategy()
	}
	return runtime.TextPart{
		Synthetic: true,
		Metadata: map[string]any{
			"codea.kind":  "model-strategy",
			"codea.level": string(strategy.Level),
		},
		Text: strategy.ControlText(),
	}
}
