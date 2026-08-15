package skill

import (
	"reflect"
	"testing"
)

func TestMergeSkillNamesInheritsDefaults(t *testing.T) {
	defaults := []string{"zebra", "apple"}
	cfg := AgentSkillConfig{
		Inherit:        true,
		Skills:         []string{"apple", "banana"},
		RequiredSkills: []SkillRequirement{{Name: "banana"}, {Name: "cherry"}},
	}
	got := MergeSkillNames(defaults, cfg)
	want := []string{"apple", "banana", "cherry", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeSkillNames = %v, want %v", got, want)
	}
}

func TestMergeSkillNamesWithoutInheritDropsDefaults(t *testing.T) {
	defaults := []string{"zebra", "apple"}
	cfg := AgentSkillConfig{
		Inherit:        false,
		Skills:         []string{"apple"},
		RequiredSkills: []SkillRequirement{{Name: "cherry"}},
	}
	got := MergeSkillNames(defaults, cfg)
	want := []string{"apple", "cherry"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeSkillNames = %v, want %v", got, want)
	}
}

func TestMergeSkillNamesDedupesAndSortsDeterministically(t *testing.T) {
	defaults := []string{"b", "a"}
	cfg := AgentSkillConfig{
		Inherit:        true,
		Skills:         []string{"c", "a"},
		RequiredSkills: []SkillRequirement{{Name: "b"}},
	}
	got := MergeSkillNames(defaults, cfg)
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeSkillNames = %v, want %v", got, want)
	}
}
