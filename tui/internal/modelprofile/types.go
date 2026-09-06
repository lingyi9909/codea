package modelprofile

import (
	"time"

	"codea/tui/internal/runtime"
)

// CapabilityLevel is Codea's conservative model capability classification.
type CapabilityLevel string

const (
	CapabilityUnknown CapabilityLevel = "unknown"
	CapabilityWeak    CapabilityLevel = "weak"
	CapabilityMedium  CapabilityLevel = "medium"
	CapabilityStrong  CapabilityLevel = "strong"
)

// ProfileVersion is the on-disk model profile schema version.
const ProfileVersion = 1

// ModelCapabilityProfile contains only safe machine qualification metadata.
type ModelCapabilityProfile struct {
	Model            runtime.ModelRef `json:"model"`
	ToolCalling      CapabilityLevel  `json:"toolCalling"`
	StructuredOutput CapabilityLevel  `json:"structuredOutput"`
	PatchFollowing   CapabilityLevel  `json:"patchFollowing"`
	Planning         CapabilityLevel  `json:"planning"`
	Overall          CapabilityLevel  `json:"overall"`
	QualifiedAt      time.Time        `json:"qualifiedAt"`
	Version          int              `json:"version"`
}

// Aggregate returns a conservative overall level. Incomplete qualification is
// kept unknown so it cannot be persisted as a completed capability profile.
func Aggregate(levels ...CapabilityLevel) CapabilityLevel {
	if len(levels) == 0 {
		return CapabilityUnknown
	}
	for _, level := range levels {
		if level == CapabilityUnknown || !validLevel(level) {
			return CapabilityUnknown
		}
	}
	for _, level := range levels {
		if level == CapabilityWeak {
			return CapabilityWeak
		}
	}
	for _, level := range levels {
		if level == CapabilityMedium {
			return CapabilityMedium
		}
	}
	return CapabilityStrong
}

func validLevel(level CapabilityLevel) bool {
	switch level {
	case CapabilityUnknown, CapabilityWeak, CapabilityMedium, CapabilityStrong:
		return true
	default:
		return false
	}
}
