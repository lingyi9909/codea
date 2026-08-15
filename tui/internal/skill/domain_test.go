package skill

import "testing"

func TestSkillSourceValid(t *testing.T) {
	known := []SkillSource{SourceCodea, SourceProject, SourceUser, SourceRuntime}
	for _, s := range known {
		if !s.Valid() {
			t.Errorf("SkillSource %q should be valid", s)
		}
	}
	if SkillSource("bogus").Valid() {
		t.Error("unknown SkillSource should be invalid")
	}
}

func TestSkillStatesIndependent(t *testing.T) {
	// Installed + Enabled but not Loaded is a legal state (a runtime load
	// failure) and must carry a diagnostic.
	s := Skill{
		Name:      "unit-test",
		Source:    SourceCodea,
		Installed: true,
		Enabled:   true,
		Loaded:    false,
		LoadError: "not reported by runtime",
	}
	if !s.Installed || !s.Enabled || s.Loaded {
		t.Fatalf("expected installed+enabled+not-loaded, got %+v", s)
	}
	if s.LoadError == "" {
		t.Fatal("load failure must carry a diagnostic")
	}
}

func TestSkillErrorFormatting(t *testing.T) {
	e := SkillError{Name: "git", Source: SourceProject, Stage: StageRequire, Message: "not loaded"}
	got := e.Error()
	want := `skill "git" (source=project) require: not loaded`
	if got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestSkillErrorImplementsError(t *testing.T) {
	var _ error = SkillError{}
}
