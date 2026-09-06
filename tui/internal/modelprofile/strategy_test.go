package modelprofile

import "testing"

func TestTask32StrategyExactMappingAndFallback(t *testing.T) {
	tests := []struct {
		name  string
		level CapabilityLevel
		want  Strategy
	}{
		{"strong", CapabilityStrong, Strategy{Level: CapabilityStrong, RepoMapMaxChars: 12000, MinPlanSteps: 3, MaxPlanSteps: 7, VerifyContinuationLimit: 2}},
		{"medium", CapabilityMedium, Strategy{Level: CapabilityMedium, RepoMapMaxChars: 8000, MinPlanSteps: 3, MaxPlanSteps: 5, VerifyContinuationLimit: 2, PreferSequentialTools: true, AvoidSubagents: true}},
		{"unknown", CapabilityUnknown, Strategy{Level: CapabilityMedium, RepoMapMaxChars: 8000, MinPlanSteps: 3, MaxPlanSteps: 5, VerifyContinuationLimit: 2, PreferSequentialTools: true, AvoidSubagents: true}},
		{"weak", CapabilityWeak, Strategy{Level: CapabilityWeak, RepoMapMaxChars: 4000, MinPlanSteps: 3, MaxPlanSteps: 3, VerifyContinuationLimit: 1, PreferSequentialTools: true, OneMutationGroupAtATime: true, AvoidSubagents: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StrategyForLevel(tt.level)
			if got != tt.want {
				t.Fatalf("StrategyForLevel(%q) = %#v, want %#v", tt.level, got, tt.want)
			}
		})
	}
}

func TestTask32StrategyControlTextIsBoundedGuidance(t *testing.T) {
	if got := StrategyForLevel(CapabilityWeak).ControlText(); got == "" {
		t.Fatal("weak strategy control text must be non-empty")
	}
	if got := StrategyForProfile(nil); got.Level != CapabilityMedium {
		t.Fatalf("missing profile fallback level = %q, want medium", got.Level)
	}
	stale := &ModelCapabilityProfile{Version: ProfileVersion - 1, Overall: CapabilityStrong}
	if got := StrategyForProfile(stale); got.Level != CapabilityMedium {
		t.Fatalf("stale profile fallback level = %q, want medium", got.Level)
	}
}
