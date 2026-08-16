package skill

import "testing"

func codea(name string, enabled bool) Skill {
	return Skill{Name: name, Source: SourceCodea, Installed: true, Enabled: enabled}
}

func TestFilterForModeCompatibleKeepsAll(t *testing.T) {
	skills := []Skill{
		codea("code-review", true),
		{Name: "proj", Source: SourceProject, Installed: true, Enabled: true},
		{Name: "user", Source: SourceUser, Installed: true, Enabled: true},
	}
	got := FilterForMode(skills, SkillPolicy{Mode: SkillModeCompatible})
	if len(got) != 3 {
		t.Fatalf("compatible must keep all, got %d", len(got))
	}
}

func TestFilterForModeStrictKeepsApprovedEnabledCodea(t *testing.T) {
	skills := []Skill{
		codea("code-review", true),
		codea("experimental", true),
		codea("unit-test", false),
		{Name: "proj", Source: SourceProject, Installed: true, Enabled: true},
		{Name: "user", Source: SourceUser, Installed: true, Enabled: true},
	}
	got := FilterForMode(skills, SkillPolicy{
		Mode:     SkillModeStrict,
		Approved: map[string]bool{"code-review": true},
	})
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if len(got) != 1 || !names["code-review"] {
		t.Fatalf("strict should keep only approved+enabled Codea: %+v", got)
	}
}

func TestFilterForModeStrictEmptyApprovedMeansAll(t *testing.T) {
	skills := []Skill{codea("code-review", true), codea("unit-test", true)}
	got := FilterForMode(skills, SkillPolicy{Mode: SkillModeStrict})
	if len(got) != 2 {
		t.Fatalf("empty approved set approves all Codea, got %d", len(got))
	}
}

func TestParseApprovedSkills(t *testing.T) {
	got := ParseApprovedSkills(" code-review , unit-test , ")
	if len(got) != 2 || !got["code-review"] || !got["unit-test"] {
		t.Fatalf("parse comma list: %v", got)
	}
	if len(ParseApprovedSkills("")) != 0 || len(ParseApprovedSkills("  ")) != 0 {
		t.Fatal("empty/whitespace must yield empty set")
	}
}

func TestStrictAllowed(t *testing.T) {
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}}
	if !p.StrictAllowed(codea("code-review", true)) {
		t.Fatal("approved Codea must be strict-allowed")
	}
	if p.StrictAllowed(codea("experimental", true)) {
		t.Fatal("unapproved Codea must not be strict-allowed")
	}
	if p.StrictAllowed(Skill{Name: "code-review", Source: SourceProject}) {
		t.Fatal("project skill must never be strict-allowed")
	}
	if p.StrictAllowed(Skill{Name: "customize-opencode", Source: SourceRuntime}) {
		t.Fatal("runtime built-in must never be strict-allowed")
	}
}
