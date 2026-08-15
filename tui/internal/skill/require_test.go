package skill

import (
	"strings"
	"testing"
)

func TestValidateRequirementsPasses(t *testing.T) {
	skills := []Skill{
		{Name: "git", Source: SourceCodea, Installed: true, Enabled: true, Loaded: true},
	}
	if err := ValidateRequirements(skills, []SkillRequirement{{Name: "git"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequirementsMissing(t *testing.T) {
	skills := []Skill{
		{Name: "git", Installed: true, Enabled: true, Loaded: true},
	}
	err := ValidateRequirements(skills, []SkillRequirement{{Name: "reviewer"}})
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
	if !strings.Contains(err.Error(), "reviewer") {
		t.Fatalf("error should name missing skill: %v", err)
	}
}

func TestValidateRequirementsDisabled(t *testing.T) {
	skills := []Skill{
		{Name: "git", Source: SourceCodea, Installed: true, Enabled: false, Loaded: false},
	}
	if err := ValidateRequirements(skills, []SkillRequirement{{Name: "git"}}); err == nil {
		t.Fatal("expected error for disabled skill")
	}
}

func TestValidateRequirementsNotLoaded(t *testing.T) {
	skills := []Skill{
		{Name: "git", Source: SourceCodea, Installed: true, Enabled: true, Loaded: false, LoadError: "not reported as loaded by runtime"},
	}
	err := ValidateRequirements(skills, []SkillRequirement{{Name: "git"}})
	if err == nil {
		t.Fatal("expected error for not-loaded skill")
	}
	if !strings.Contains(err.Error(), "not loaded") && !strings.Contains(err.Error(), "not reported") {
		t.Fatalf("error should carry load reason: %v", err)
	}
}
