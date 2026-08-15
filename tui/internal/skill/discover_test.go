package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, root, name, frontmatter string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(frontmatter), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverFindsSkillsByDirectory(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "git", "---\nname: git\ndescription: Git helpers.\n---\nbody")
	writeSkill(t, root, "code-review", "---\nname: code-review\ndescription: Review code.\n---\nbody")

	skills, errs := Discover([]Root{{Dir: root, Source: SourceCodea}})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d: %+v", len(skills), skills)
	}
	if skills[0].Name != "code-review" || skills[1].Name != "git" {
		t.Fatalf("skills not sorted by name: %+v", skills)
	}
	for _, s := range skills {
		if !s.Installed || s.Source != SourceCodea || s.Description == "" {
			t.Errorf("bad skill state: %+v", s)
		}
	}
}

func TestDiscoverSourceAssignment(t *testing.T) {
	codea := t.TempDir()
	project := t.TempDir()
	writeSkill(t, codea, "builtin-a", "---\ndescription: d\n---\n")
	writeSkill(t, project, "proj-a", "---\ndescription: d\n---\n")

	skills, _ := Discover([]Root{
		{Dir: codea, Source: SourceCodea},
		{Dir: project, Source: SourceProject},
	})
	src := map[string]SkillSource{}
	for _, s := range skills {
		src[s.Name] = s.Source
	}
	if src["builtin-a"] != SourceCodea || src["proj-a"] != SourceProject {
		t.Fatalf("wrong source assignment: %v", src)
	}
}

func TestDiscoverBrokenSkillIsolated(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "good", "---\nname: good\ndescription: ok\n---\n")
	// Make SKILL.md a directory so os.ReadFile fails cross-platform.
	broken := filepath.Join(root, "broken")
	if err := os.MkdirAll(filepath.Join(broken, "SKILL.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	skills, errs := Discover([]Root{{Dir: root, Source: SourceProject}})
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills (good + broken), got %d: %+v", len(skills), skills)
	}
	if len(errs) != 1 || errs[0].Name != "broken" || errs[0].Stage != StageDiscover {
		t.Fatalf("expected one discover error for broken, got %v", errs)
	}
}

func TestDiscoverMissingRootSkipped(t *testing.T) {
	skills, errs := Discover([]Root{{Dir: filepath.Join(t.TempDir(), "nope"), Source: SourceUser}})
	if len(skills) != 0 || len(errs) != 0 {
		t.Fatalf("missing root should be skipped silently, got skills=%v errs=%v", skills, errs)
	}
}

func TestDiscoverIgnoresDirWithoutSkill(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "not-a-skill"), 0o755); err != nil {
		t.Fatal(err)
	}
	skills, errs := Discover([]Root{{Dir: root, Source: SourceCodea}})
	if len(skills) != 0 || len(errs) != 0 {
		t.Fatalf("dir without SKILL.md must be ignored, got skills=%v errs=%v", skills, errs)
	}
}

func TestDiscoverFrontmatterFallback(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "no-frontmatter", "just a body, no frontmatter")

	skills, _ := Discover([]Root{{Dir: root, Source: SourceCodea}})
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %+v", skills)
	}
	if skills[0].Name != "no-frontmatter" || skills[0].Description != "" {
		t.Fatalf("name should fall back to dir, description empty: %+v", skills[0])
	}
}

func TestParseFrontmatter(t *testing.T) {
	cases := []struct {
		body        string
		name, desc  string
	}{
		{"---\nname: x\ndescription: y\n---\n", "x", "y"},
		{"---\nname: \"quoted\"\ndescription: 'single'\n---\n", "quoted", "single"},
		{"no frontmatter", "", ""},
		{"---\ndescription: only-desc\n---\n", "", "only-desc"},
	}
	for _, c := range cases {
		n, d := parseFrontmatter([]byte(c.body))
		if n != c.name || d != c.desc {
			t.Errorf("parseFrontmatter(%q) = (%q, %q), want (%q, %q)", c.body, n, d, c.name, c.desc)
		}
	}
}
