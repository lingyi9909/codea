package skill

import (
	"strings"
	"testing"
)

func TestParseSkillModeKnown(t *testing.T) {
	for _, in := range []string{"strict", "compatible"} {
		m, err := ParseSkillMode(in)
		if err != nil {
			t.Fatalf("ParseSkillMode(%q): %v", in, err)
		}
		if string(m) != in {
			t.Fatalf("ParseSkillMode(%q) = %q", in, m)
		}
	}
}

func TestParseSkillModeUnknownErrors(t *testing.T) {
	if _, err := ParseSkillMode("bogus"); err == nil {
		t.Fatal("unknown mode must error, not silently compatible")
	} else if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("error should name the bad value: %v", err)
	}
}

func TestResolveSkillModeDefaultsStrict(t *testing.T) {
	m, err := ResolveSkillMode("")
	if err != nil {
		t.Fatalf("empty should default, not error: %v", err)
	}
	if m != SkillModeStrict {
		t.Fatalf("default = %q, want strict", m)
	}
}

func TestSkillModeValid(t *testing.T) {
	if !SkillModeStrict.Valid() || !SkillModeCompatible.Valid() {
		t.Fatal("strict and compatible must be valid")
	}
	if SkillMode("").Valid() || SkillMode("nope").Valid() {
		t.Fatal("empty/unknown must be invalid")
	}
}
