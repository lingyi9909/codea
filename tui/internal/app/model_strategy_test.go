package app

import (
	"context"
	"strings"
	"testing"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/repoctx"
	"codea/tui/internal/runtime"
)

type task32BudgetRepoService struct{ got repoctx.Query }
func (f *task32BudgetRepoService) BuildMap(_ context.Context, q repoctx.Query) (repoctx.RepoMap, error) { f.got = q; return repoMapFixture(), nil }
func (f *task32BudgetRepoService) Invalidate() {}

func TestTask32SelectedModelProfileNeverLeaksAcrossModelSwitch(t *testing.T) {
	store := modelprofile.NewStore(t.TempDir())
	a := runtime.ModelRef{ProviderID: "private", ModelID: "a"}
	b := runtime.ModelRef{ProviderID: "private", ModelID: "b"}
	for ref, level := range map[runtime.ModelRef]modelprofile.CapabilityLevel{a: modelprofile.CapabilityStrong, b: modelprofile.CapabilityWeak} {
		p := modelprofile.ModelCapabilityProfile{Model: ref, ToolCalling: level, StructuredOutput: level, PatchFollowing: level, Planning: level, Overall: level, Version: modelprofile.ProfileVersion}
		if err := store.Save(p); err != nil { t.Fatal(err) }
	}
	m := NewModel(nil)
	m.SetModelProfileStore(store)
	m.sessionID = "s"
	m.sessionModels[m.sessionID] = a
	if got := m.currentModelStrategy(runtime.PromptRequest{}); got.Level != modelprofile.CapabilityStrong { t.Fatalf("A strategy = %q", got.Level) }
	m.sessionModels[m.sessionID] = b
	if got := m.currentModelStrategy(runtime.PromptRequest{}); got.Level != modelprofile.CapabilityWeak { t.Fatalf("B strategy = %q; A profile leaked or B ignored", got.Level) }
}

func TestTask32RuntimeDefaultProfileUsedOnlyWhenExactlyResolvable(t *testing.T) {
	store := modelprofile.NewStore(t.TempDir())
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "default"}
	p := modelprofile.ModelCapabilityProfile{Model: ref, ToolCalling: modelprofile.CapabilityStrong, StructuredOutput: modelprofile.CapabilityStrong, PatchFollowing: modelprofile.CapabilityStrong, Planning: modelprofile.CapabilityStrong, Overall: modelprofile.CapabilityStrong, Version: modelprofile.ProfileVersion}
	if err := store.Save(p); err != nil { t.Fatal(err) }
	m := NewModel(nil); m.SetModelProfileStore(store)
	m.runtimeModels = []runtime.Model{{Ref: ref, Default: true}}
	if got := m.currentModelStrategy(runtime.PromptRequest{}); got.Level != modelprofile.CapabilityStrong { t.Fatalf("default strategy = %q", got.Level) }
	m.runtimeModels = append(m.runtimeModels, runtime.Model{Ref: runtime.ModelRef{ProviderID:"private", ModelID:"other"}, Default:true})
	if got := m.currentModelStrategy(runtime.PromptRequest{}); got.Level != modelprofile.CapabilityMedium { t.Fatalf("ambiguous default strategy = %q, want medium", got.Level) }
}

func TestTask32RepoMapBudgetAndPromptPartOrder(t *testing.T) {
	strategy := modelprofile.StrategyForLevel(modelprofile.CapabilityStrong)
	service := &task32BudgetRepoService{}
	intent := repoPromptIntent{request: runtime.PromptRequest{Agent:"general"}, promptText:"fix it", queryText:"fix it", strategy: strategy}
	msg := RepoContextCmd(service, intent)().(repoContextResultMsg)
	if service.got.MaxChars != 12000 { t.Fatalf("repo MaxChars = %d, want 12000", service.got.MaxChars) }
	req := buildRepoAwarePrompt(intent, msg.mapOut, nil)
	if len(req.Parts) != 4 { t.Fatalf("parts = %#v, want model-strategy, repo-map, task-strategy, user", req.Parts) }
	wantKinds := []string{"model-strategy","repo-map","task-strategy"}
	for i, want := range wantKinds {
		p, ok := req.Parts[i].(runtime.TextPart)
		if !ok || !p.Synthetic || p.Metadata["codea.kind"] != want { t.Fatalf("parts[%d] = %#v, want %s", i, req.Parts[i], want) }
	}
	user := req.Parts[3].(runtime.TextPart)
	if user.Synthetic || user.Text != "fix it" { t.Fatalf("user part = %#v", user) }
}

func TestTask32PlanningGuidanceKeepsMachineRangeAndAdaptsPreference(t *testing.T) {
	weak := modelprofile.StrategyForLevel(modelprofile.CapabilityWeak)
	part, ok := taskStrategyPart("general", weak)
	if !ok || !strings.Contains(part.Text, "3–7") || !strings.Contains(part.Text, "exactly 3") { t.Fatalf("weak task strategy = %#v", part) }
	medium := modelprofile.StrategyForLevel(modelprofile.CapabilityMedium)
	part, _ = taskStrategyPart("general", medium)
	if !strings.Contains(part.Text, "3–5") { t.Fatalf("medium task strategy = %q", part.Text) }
}

func TestTask32VerificationBudgetFreezesAtRootTaskStart(t *testing.T) {
	m := NewModel(nil)
	m.activeModelStrategy = modelprofile.StrategyForLevel(modelprofile.CapabilityWeak)
	m.resetTaskExecution("root")
	if m.taskExecution.VerificationContinuationLimit != 1 { t.Fatalf("weak root limit = %d", m.taskExecution.VerificationContinuationLimit) }
	m.activeModelStrategy = modelprofile.StrategyForLevel(modelprofile.CapabilityStrong)
	if m.taskExecution.VerificationContinuationLimit != 1 { t.Fatal("same root verification limit changed after strategy/model switch") }
	m.resetTaskExecution("next")
	if m.taskExecution.VerificationContinuationLimit != 2 { t.Fatalf("next root strong limit = %d", m.taskExecution.VerificationContinuationLimit) }
}
