package app

import (
	"errors"
	"os"
	"testing"
	"time"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"
)

func TestTask32QualificationAcceptanceScenarios(t *testing.T) {
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "coder"}
	cases := []struct {
		name string
		state ModelCheckState
		want modelprofile.CapabilityLevel
	}{
		{
			name: "strong all first pass",
			state: ModelCheckState{Model: ref,
				ToolCalling: ProbeOutcome{Attempts: 1, Passed: true},
				Structured: ProbeOutcome{Attempts: 1, Passed: true},
				Patch: ProbeOutcome{Attempts: 1, Passed: true},
				Planning: ProbeOutcome{Attempts: 1, Passed: true}},
			want: modelprofile.CapabilityStrong,
		},
		{
			name: "medium mixed first and second pass",
			state: ModelCheckState{Model: ref,
				ToolCalling: ProbeOutcome{Attempts: 1, Passed: true},
				Structured: ProbeOutcome{Attempts: 2, Passed: true},
				Patch: ProbeOutcome{Attempts: 1, Passed: true},
				Planning: ProbeOutcome{Attempts: 1, Passed: true}},
			want: modelprofile.CapabilityMedium,
		},
		{
			name: "weak any probe fails twice",
			state: ModelCheckState{Model: ref,
				ToolCalling: ProbeOutcome{Attempts: 1, Passed: true},
				Structured: ProbeOutcome{Attempts: 1, Passed: true},
				Patch: ProbeOutcome{Attempts: 2, Passed: false},
				Planning: ProbeOutcome{Attempts: 1, Passed: true}},
			want: modelprofile.CapabilityWeak,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &Model{modelCheck: tc.state}
			got := m.completedModelProfile(time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC))
			if got.Overall != tc.want {
				t.Fatalf("overall=%q want=%q profile=%#v", got.Overall, tc.want, got)
			}
			if strategy := modelprofile.StrategyForProfile(&got); strategy.Level != tc.want {
				t.Fatalf("strategy=%q want=%q", strategy.Level, tc.want)
			}
		})
	}
}

func TestTask32AbortedQualificationPersistsNothingAndFallsBackMedium(t *testing.T) {
	home := t.TempDir()
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "aborted"}
	m := &Model{
		modelProfileStore: modelprofile.NewStore(home),
		modelCheck: ModelCheckState{Active: true, Model: ref},
	}
	m.failModelCheck("qualification interrupted")
	if m.modelCheck.Active || m.modelCheck.Terminal {
		t.Fatalf("aborted state=%#v", m.modelCheck)
	}
	if _, err := os.Stat(modelprofile.ProfilePath(home, ref)); !os.IsNotExist(err) {
		t.Fatalf("aborted qualification persisted a profile: %v", err)
	}
	if got := modelprofile.StrategyForProfile(nil).Level; got != modelprofile.CapabilityMedium {
		t.Fatalf("fallback=%q want medium", got)
	}
}

func TestTask32AdaptiveStrategyPreservesAgentRouteAndVerificationTruth(t *testing.T) {
	verification := TaskExecutionState{MutationSeen: true, VerifyPassed: true, LastVerificationResult: "pass"}
	for _, level := range []modelprofile.CapabilityLevel{modelprofile.CapabilityStrong, modelprofile.CapabilityMedium, modelprofile.CapabilityWeak} {
		strategy := modelprofile.StrategyForLevel(level)
		intent := repoPromptIntent{
			request: runtime.PromptRequest{MessageID: "m1", Agent: "general"},
			promptText: "Fix OrderService",
			queryText: "Fix OrderService",
			strategy: strategy,
		}
		req := buildRepoAwarePrompt(intent, repoMapFixture(), errors.New("force no repo-map; strategy must still preserve route"))
		if req.Agent != "general" {
			t.Fatalf("level=%q changed agent route to %q", level, req.Agent)
		}
		if got := verificationDecision(verification); got != VerifyAccepted {
			t.Fatalf("level=%q changed verification truth to %q", level, got)
		}
	}
}
