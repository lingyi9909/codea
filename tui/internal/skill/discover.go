package skill

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Root describes a filesystem directory to scan for skills and the source to
// assign to any skill discovered there.
type Root struct {
	Dir    string
	Source SkillSource
}

// Discover scans roots for <name>/SKILL.md entries and returns the discovered
// skills plus any per-skill diagnostics. A failure on one skill or root never
// aborts the scan of the others; results are sorted deterministically.
func Discover(roots []Root) ([]Skill, []SkillError) {
	var skills []Skill
	var errs []SkillError

	for _, root := range roots {
		entries, err := os.ReadDir(root.Dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			errs = append(errs, SkillError{
				Name:    root.Dir,
				Source:  root.Source,
				Stage:   StageDiscover,
				Message: err.Error(),
			})
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dirName := entry.Name()
			skillDir := filepath.Join(root.Dir, dirName)
			body, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
			if err != nil {
				if os.IsNotExist(err) {
					continue // directory without SKILL.md is not a skill
				}
				skills = append(skills, Skill{Name: dirName, Source: root.Source, Installed: true, dir: skillDir})
				errs = append(errs, SkillError{
					Name:    dirName,
					Source:  root.Source,
					Stage:   StageDiscover,
					Message: "read SKILL.md: " + err.Error(),
				})
				continue
			}
			name, description := parseFrontmatter(body)
			if name == "" {
				name = dirName
			}
			skills = append(skills, Skill{
				Name:        name,
				Description: description,
				Source:      root.Source,
				Installed:   true,
				dir:         skillDir,
			})
		}
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Source < skills[j].Source
	})
	sort.Slice(errs, func(i, j int) bool {
		if errs[i].Name != errs[j].Name {
			return errs[i].Name < errs[j].Name
		}
		return errs[i].Stage < errs[j].Stage
	})
	return skills, errs
}

// parseFrontmatter extracts the name and description keys from a SKILL.md
// frontmatter block (a leading "---" fence terminated by another "---"). It
// returns empty strings when the block is absent or a key is missing.
func parseFrontmatter(body []byte) (name, description string) {
	lines := strings.Split(string(body), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", ""
	}
	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			break
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		switch strings.TrimSpace(key) {
		case "name":
			name = value
		case "description":
			description = value
		}
	}
	return name, description
}
