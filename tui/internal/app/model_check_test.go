package app

import (
	"testing"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"
)

func TestTask32QualificationScoringIsConservative(t *testing.T) {
	tests := []struct{ name string; in ProbeOutcome; want modelprofile.CapabilityLevel }{
		{"first pass", ProbeOutcome{Attempts: 1, Passed: true}, modelprofile.CapabilityStrong},
		{"retry pass", ProbeOutcome{Attempts: 2, Passed: true}, modelprofile.CapabilityMedium},
		{"two failures", ProbeOutcome{Attempts: 2}, modelprofile.CapabilityWeak},
		{"missing", ProbeOutcome{}, modelprofile.CapabilityWeak},
		{"one failure at terminal", ProbeOutcome{Attempts: 1}, modelprofile.CapabilityWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := levelForProbe(tt.in); got != tt.want { t.Fatalf("levelForProbe(%#v) = %q, want %q", tt.in, got, tt.want) }
		})
	}
}

func TestTask32ModelCheckPromptIsSyntheticAndExactModel(t *testing.T) {
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "coder-v2"}
	req := modelCheckPrompt(ref)
	if req.Agent != "model-evaluator" || req.Model == nil || *req.Model != ref { t.Fatalf("request identity = %#v", req) }
	if len(req.Parts) != 1 { t.Fatalf("parts = %d, want 1", len(req.Parts)) }
	part, ok := req.Parts[0].(runtime.TextPart)
	if !ok || !part.Synthetic || part.Metadata["codea.kind"] != "model-check" || part.Text != fixedQualificationInstruction {
		t.Fatalf("qualification part = %#v", req.Parts[0])
	}
}

func TestTask32InternalSessionsAreHidden(t *testing.T) {
	sessions := []runtime.Session{
		{ID: "user-1", Title: "Feature work"},
		{ID: "internal-1", Title: internalModelCheckTitlePrefix + "private/coder"},
		{ID: "user-2", Title: "Review"},
	}
	got := filterInternalSessions(sessions)
	if len(got) != 2 || got[0].ID != "user-1" || got[1].ID != "user-2" { t.Fatalf("filtered sessions = %#v", got) }
}

func TestTask32OnlyExactInternalSessionProbeEventsScore(t *testing.T) {
	m := &Model{modelCheck: ModelCheckState{Active: true, SessionID: "internal", Model: runtime.ModelRef{ProviderID: "p", ModelID: "m"}}}
	foreign := runtime.Event{SessionID: "user", Tool: &runtime.ToolEvent{Metadata: map[string]string{"codeaProbe": "state", "codeaProbeResult": "pass", "codeaProbeAttempt": "1"}}}
	if handled, _ := m.handleModelCheckEvent(foreign); handled { t.Fatal("foreign session event was intercepted") }
	if m.modelCheck.ToolCalling.Attempts != 0 { t.Fatal("foreign event changed qualification state") }
	own := runtime.Event{SessionID: "internal", Tool: &runtime.ToolEvent{Metadata: map[string]string{"codeaProbe": "state", "codeaProbeResult": "pass", "codeaProbeAttempt": "1"}}}
	if handled, _ := m.handleModelCheckEvent(own); !handled { t.Fatal("internal session event was not intercepted") }
	if m.modelCheck.ToolCalling != (ProbeOutcome{Attempts: 1, Passed: true}) { t.Fatalf("outcome = %#v", m.modelCheck.ToolCalling) }
}

func TestTask32UniqueRuntimeDefaultResolution(t *testing.T) {
	models := []runtime.Model{{Ref: runtime.ModelRef{ProviderID: "p", ModelID: "a"}, Default: true}, {Ref: runtime.ModelRef{ProviderID: "p", ModelID: "b"}}}
	ref, ok := uniqueDefaultModel(models)
	if !ok || ref.ModelID != "a" { t.Fatalf("unique default = %#v, %t", ref, ok) }
	models[1].Default = true
	if _, ok := uniqueDefaultModel(models); ok { t.Fatal("multiple defaults must be ambiguous") }
}
