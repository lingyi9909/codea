package parity

import "testing"

func TestRequiredReleaseGateIDsCoversG1ThroughG15IncludingSubgates(t *testing.T) {
	want := []string{"G1", "G2", "G2.1", "G3", "G4", "G5", "G6", "G7", "G8", "G9", "G10", "G11", "G12", "G12.1", "G13", "G14", "G15"}
	got := RequiredReleaseGateIDs()
	if len(got) != len(want) {
		t.Fatalf("gate count=%d want=%d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("gate[%d]=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestCertificationValidateRequiresEveryGateAndParity(t *testing.T) {
	gates := make([]ReleaseGateEvidence, 0, len(RequiredReleaseGateIDs()))
	for _, id := range RequiredReleaseGateIDs() {
		gates = append(gates, ReleaseGateEvidence{ID: id, Status: ReleaseGatePass, Evidence: "fresh"})
	}
	cert := Certification{
		SourceCommit:          "abc123",
		Gates:                 gates,
		Parity:                &Result{Total: 12, Passed: 12, RequiredFailed: 0},
		GeneralCompletionRate: 1.0,
	}
	if err := cert.Validate(); err != nil {
		t.Fatalf("all-pass certification rejected: %v", err)
	}

	missing := cert
	missing.Gates = missing.Gates[:len(missing.Gates)-1]
	if err := missing.Validate(); err == nil {
		t.Fatal("missing required gate must fail certification")
	}

	deferred := cert
	deferred.Gates = append([]ReleaseGateEvidence(nil), cert.Gates...)
	deferred.Gates[0].Status = ReleaseGateDeferred
	if err := deferred.Validate(); err == nil {
		t.Fatal("deferred required release gate must fail final certification")
	}

	lowParity := cert
	lowParity.GeneralCompletionRate = 0.94
	if err := lowParity.Validate(); err == nil {
		t.Fatal("general completion rate below 95% must fail certification")
	}

	requiredFailure := cert
	requiredFailure.Parity = &Result{Total: 12, Passed: 11, Failed: 1, RequiredFailed: 1}
	if err := requiredFailure.Validate(); err == nil {
		t.Fatal("required parity failure must fail certification")
	}
}

func TestCertificationRejectsDuplicateAndUnknownGateIDs(t *testing.T) {
	gates := make([]ReleaseGateEvidence, 0, len(RequiredReleaseGateIDs()))
	for _, id := range RequiredReleaseGateIDs() {
		gates = append(gates, ReleaseGateEvidence{ID: id, Status: ReleaseGatePass, Evidence: "fresh"})
	}
	base := Certification{
		SourceCommit:          "abc123",
		Gates:                 gates,
		Parity:                &Result{Total: 12, Passed: 12},
		GeneralCompletionRate: 1.0,
	}

	duplicate := base
	duplicate.Gates = append(append([]ReleaseGateEvidence(nil), gates...), gates[0])
	if err := duplicate.Validate(); err == nil {
		t.Fatal("duplicate gate must fail certification")
	}

	unknown := base
	unknown.Gates = append([]ReleaseGateEvidence(nil), gates...)
	unknown.Gates[0].ID = "G99"
	if err := unknown.Validate(); err == nil {
		t.Fatal("unknown gate must fail certification")
	}
}
