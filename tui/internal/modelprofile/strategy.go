package modelprofile

import "fmt"

// Strategy is the deterministic Codea control policy selected from a model's
// qualified capability profile. It is advisory for model behavior; existing
// machine safety gates remain authoritative.
type Strategy struct {
	Level                   CapabilityLevel
	RepoMapMaxChars         int
	MinPlanSteps            int
	MaxPlanSteps            int
	VerifyContinuationLimit int
	PreferSequentialTools   bool
	OneMutationGroupAtATime bool
	AvoidSubagents          bool
}

func StrategyForLevel(level CapabilityLevel) Strategy {
	switch level {
	case CapabilityStrong:
		return Strategy{Level: CapabilityStrong, RepoMapMaxChars: 12000, MinPlanSteps: 3, MaxPlanSteps: 7, VerifyContinuationLimit: 2}
	case CapabilityWeak:
		return Strategy{Level: CapabilityWeak, RepoMapMaxChars: 4000, MinPlanSteps: 3, MaxPlanSteps: 3, VerifyContinuationLimit: 1, PreferSequentialTools: true, OneMutationGroupAtATime: true, AvoidSubagents: true}
	case CapabilityMedium, CapabilityUnknown:
		fallthrough
	default:
		return Strategy{Level: CapabilityMedium, RepoMapMaxChars: 8000, MinPlanSteps: 3, MaxPlanSteps: 5, VerifyContinuationLimit: 2, PreferSequentialTools: true, AvoidSubagents: true}
	}
}

// StrategyForProfile deliberately falls back to medium for absent, stale, or
// invalid profile state. A missing qualification must never select strong.
func StrategyForProfile(profile *ModelCapabilityProfile) Strategy {
	if profile == nil || profile.Version != ProfileVersion || !validLevel(profile.Overall) || profile.Overall == CapabilityUnknown {
		return StrategyForLevel(CapabilityMedium)
	}
	return StrategyForLevel(profile.Overall)
}

// ControlText is bounded public strategy guidance. It contains no provider
// payload, model output, or chain-of-thought.
func (s Strategy) ControlText() string {
	return fmt.Sprintf("Codea model strategy: level=%s; repo-map-max-chars=%d; plan-steps=%d..%d; verification-continuations=%d; prefer-sequential-tools=%t; one-mutation-group-at-a-time=%t; avoid-subagents=%t. Existing Codea safety, approval, DLP, project-boundary, planning, verification, and checkpoint gates remain authoritative.",
		s.Level, s.RepoMapMaxChars, s.MinPlanSteps, s.MaxPlanSteps, s.VerifyContinuationLimit, s.PreferSequentialTools, s.OneMutationGroupAtATime, s.AvoidSubagents)
}
