package app

import (
	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"
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
