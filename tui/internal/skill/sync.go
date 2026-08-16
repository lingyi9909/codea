package skill

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// SyncEnabled discovers skills, applies the persisted overrides and the mode
// policy, and materializes the resulting enabled Codea skills into targetDir.
// It is the cold-start equivalent of Manager.SetEnabled's sync path and
// deliberately does not query the runtime, so it can run before the runtime
// process has started.
func SyncEnabled(roots []Root, store Store, targetDir string, p SkillPolicy) error {
	skills, _ := Discover(roots)
	overrides, err := store.Load()
	if err != nil {
		return fmt.Errorf("load skill overrides: %w", err)
	}
	skills = applyOverrides(skills, overrides)
	skills = FilterForMode(skills, p)
	return Sync(skills, targetDir)
}

// Sync materializes enabled Codea skills into targetDir (the controlled runtime
// config directory) and removes any previously synced skill that is no longer
// enabled. Only Source=Codea skills are copied; project/user skills are
// discovered by the runtime directly and are not managed here.
func Sync(skills []Skill, targetDir string) error {
	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	sorted := append([]Skill(nil), skills...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	for _, s := range sorted {
		if s.Source != SourceCodea || !s.Enabled || s.dir == "" {
			continue
		}
		if err := copyDir(s.dir, filepath.Join(targetDir, filepath.Base(s.dir))); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		s, d := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
			continue
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		if err := copyFile(s, d, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
