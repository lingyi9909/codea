package skill

import "testing"

func TestReservedProfiles(t *testing.T) {
	if ProfileGeneral.SkillMode != SkillModeCompatible {
		t.Fatalf("general must be compatible, got %q", ProfileGeneral.SkillMode)
	}
	if ProfileEnterprise.SkillMode != SkillModeStrict {
		t.Fatalf("enterprise must be strict, got %q", ProfileEnterprise.SkillMode)
	}
}

func TestProfileValidateEnterpriseDowngradeRejected(t *testing.T) {
	p := Profile{Name: "enterprise", SkillMode: SkillModeCompatible}
	if err := p.Validate(); err == nil {
		t.Fatal("enterprise must not be downgradable to compatible")
	}
}

func TestProfileValidateOK(t *testing.T) {
	if err := ProfileEnterprise.Validate(); err != nil {
		t.Fatalf("enterprise profile invalid: %v", err)
	}
	if err := ProfileGeneral.Validate(); err != nil {
		t.Fatalf("general profile invalid: %v", err)
	}
}
