package skill

import (
	"strings"
	"testing"
)

func TestValidateSkillOK(t *testing.T) {
	if err := ValidateSkill(Skill{Name: "code-review", Source: SourceCodea}); err != nil {
		t.Fatalf("valid skill rejected: %v", err)
	}
}

func TestValidateSkillEmptyName(t *testing.T) {
	err := ValidateSkill(Skill{Name: "  ", Source: SourceProject})
	if err == nil {
		t.Fatal("empty name must error")
	}
	var se SkillError
	if !asSkillError(err, &se) || se.Stage != StageDiscover {
		t.Fatalf("expected a discover-stage SkillError, got %v", err)
	}
}

func TestValidateSkillUnknownSource(t *testing.T) {
	err := ValidateSkill(Skill{Name: "x", Source: SkillSource("mars")})
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("unknown source must error clearly: %v", err)
	}
}

func asSkillError(err error, out *SkillError) bool {
	se, ok := err.(SkillError)
	if ok {
		*out = se
	}
	return ok
}
