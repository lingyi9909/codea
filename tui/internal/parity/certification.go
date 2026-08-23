package parity

import "fmt"

type ReleaseGateStatus string

const (
	ReleaseGatePass     ReleaseGateStatus = "pass"
	ReleaseGateFail     ReleaseGateStatus = "fail"
	ReleaseGateDeferred ReleaseGateStatus = "deferred"
)

type ReleaseGateEvidence struct {
	ID       string            `json:"id"`
	Status   ReleaseGateStatus `json:"status"`
	Evidence string            `json:"evidence"`
}

// Certification is the machine-checkable Task 21 release verdict. Final V1
// certification is stricter than earlier task-level deferrals: every G1-G15
// gate (including G2.1 and G12.1) must have fresh PASS evidence.
type Certification struct {
	SourceCommit          string                `json:"sourceCommit"`
	Gates                 []ReleaseGateEvidence `json:"gates"`
	Parity                *Result               `json:"parity"`
	GeneralCompletionRate float64               `json:"generalCompletionRate"`
}

var requiredReleaseGateIDs = []string{
	"G1", "G2", "G2.1", "G3", "G4", "G5", "G6", "G7", "G8", "G9", "G10",
	"G11", "G12", "G12.1", "G13", "G14", "G15",
}

func RequiredReleaseGateIDs() []string {
	return append([]string(nil), requiredReleaseGateIDs...)
}

func (c Certification) Validate() error {
	if c.SourceCommit == "" {
		return fmt.Errorf("release certification sourceCommit is required")
	}
	if c.Parity == nil || c.Parity.Total <= 0 {
		return fmt.Errorf("release parity result is required")
	}
	if c.Parity.RequiredFailed != 0 {
		return fmt.Errorf("release parity has %d required failure(s)", c.Parity.RequiredFailed)
	}
	if c.GeneralCompletionRate < 0.95 {
		return fmt.Errorf("general completion rate %.4f is below 0.95", c.GeneralCompletionRate)
	}

	required := make(map[string]struct{}, len(requiredReleaseGateIDs))
	for _, id := range requiredReleaseGateIDs {
		required[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(c.Gates))
	for _, gate := range c.Gates {
		if _, ok := required[gate.ID]; !ok {
			return fmt.Errorf("unknown release gate %q", gate.ID)
		}
		if _, duplicate := seen[gate.ID]; duplicate {
			return fmt.Errorf("duplicate release gate %q", gate.ID)
		}
		seen[gate.ID] = struct{}{}
		if gate.Status != ReleaseGatePass {
			return fmt.Errorf("release gate %s is %s, want pass", gate.ID, gate.Status)
		}
		if gate.Evidence == "" {
			return fmt.Errorf("release gate %s has no evidence", gate.ID)
		}
	}
	for _, id := range requiredReleaseGateIDs {
		if _, ok := seen[id]; !ok {
			return fmt.Errorf("required release gate %s is missing", id)
		}
	}
	return nil
}
