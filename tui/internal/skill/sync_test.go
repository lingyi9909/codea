package skill

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
)

func writeSkillDir(t *testing.T, root, name string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: d\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSyncCopiesOnlyEnabledCodea(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "enabled")
	writeSkillDir(t, root, "disabled")

	skills := []Skill{
		{Name: "enabled", Source: SourceCodea, Enabled: true, dir: filepath.Join(root, "enabled")},
		{Name: "disabled", Source: SourceCodea, Enabled: false, dir: filepath.Join(root, "disabled")},
	}
	target := filepath.Join(t.TempDir(), "skills")
	if err := Sync(skills, target); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(target, "enabled", "SKILL.md")); err != nil {
		t.Fatalf("enabled skill not synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "disabled")); !os.IsNotExist(err) {
		t.Fatalf("disabled skill must not be synced: %v", err)
	}
}

func TestSyncRemovesStale(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "current")
	target := filepath.Join(t.TempDir(), "skills")

	// Pre-seed a stale skill in the target.
	stale := filepath.Join(target, "stale")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}

	skills := []Skill{
		{Name: "current", Source: SourceCodea, Enabled: true, dir: filepath.Join(root, "current")},
	}
	if err := Sync(skills, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale skill must be removed: %v", err)
	}
}

func TestSyncSkipsNonCodea(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "project-skill")
	target := filepath.Join(t.TempDir(), "skills")

	skills := []Skill{
		{Name: "project-skill", Source: SourceProject, Enabled: true, dir: filepath.Join(root, "project-skill")},
	}
	if err := Sync(skills, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "project-skill")); !os.IsNotExist(err) {
		t.Fatalf("non-Codea skill must not be synced: %v", err)
	}
}

func TestSyncCopiesWholeDir(t *testing.T) {
	root := t.TempDir()
	dir := writeSkillDir(t, root, "with-script")
	if err := os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	skills := []Skill{{Name: "with-script", Source: SourceCodea, Enabled: true, dir: dir}}
	target := filepath.Join(t.TempDir(), "skills")
	if err := Sync(skills, target); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(target, "with-script", "run.sh"))
	if err != nil {
		t.Fatalf("extra file not copied: %v", err)
	}
	// Windows does not expose/preserve the Unix executable bit. The whole-file
	// copy remains required; executable-mode preservation is asserted on Unix.
	if goruntime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("executable bit not preserved: %v", info.Mode())
	}
}
