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

func TestValidateRequiredSkillsStrictNotApproved(t *testing.T) {
	skills := []Skill{
		{Name: "enterprise-review", Source: SourceCodea, Installed: true, Enabled: true, Loaded: true},
	}
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"other": true}}
	err := ValidateRequiredSkills(skills, []SkillRequirement{{Name: "enterprise-review"}}, p)
	if err == nil {
		t.Fatal("required skill not approved must fail in strict mode")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("error should say not-allowed: %v", err)
	}
}

func TestValidateRequiredSkillsStrictProjectSkillRejected(t *testing.T) {
	skills := []Skill{
		{Name: "proj", Source: SourceProject, Installed: true, Enabled: true, Loaded: true},
	}
	p := SkillPolicy{Mode: SkillModeStrict} // empty approved = all Codea approved, but proj is not Codea
	err := ValidateRequiredSkills(skills, []SkillRequirement{{Name: "proj"}}, p)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("project skill must fail strict requiredSkills: %v", err)
	}
}

func TestValidateRequiredSkillsStrictPass(t *testing.T) {
	skills := []Skill{
		{Name: "code-review", Source: SourceCodea, Installed: true, Enabled: true, Loaded: true},
	}
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}}
	if err := ValidateRequiredSkills(skills, []SkillRequirement{{Name: "code-review"}}, p); err != nil {
		t.Fatalf("approved+enabled+loaded must pass: %v", err)
	}
}
