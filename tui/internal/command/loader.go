package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadWorkspaceRegistry loads command layers in the approved priority order:
// Built-in > Enterprise > Project. Register is fail-closed, so any lower-layer
// name/alias collision returns COMMAND_CONFLICT instead of silently overriding.
func LoadWorkspaceRegistry(enterpriseDir, projectDir string) (*Registry, error) {
	reg := NewRegistry()
	for _, def := range BuiltinCommands() {
		if err := reg.Register(def); err != nil {
			return nil, err
		}
	}
	if err := loadDirectory(reg, enterpriseDir, SourceEnterprise); err != nil {
		return nil, err
	}
	if err := loadDirectory(reg, projectDir, SourceProject); err != nil {
		return nil, err
	}
	return reg, nil
}

func loadDirectory(reg *Registry, dir string, source Source) error {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return &Error{Code: CodeInvalid, Detail: fmt.Sprintf("read %s command directory %s: %v", source, dir, err)}
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return &Error{Code: CodeInvalid, Command: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), Detail: fmt.Sprintf("read %s: %v", path, err)}
		}
		def, err := parseMarkdownDefinition(entry.Name(), string(data), source)
		if err != nil {
			return err
		}
		if err := reg.Register(def); err != nil {
			return err
		}
	}
	return nil
}

func parseMarkdownDefinition(filename, raw string, source Source) (Definition, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	meta := make(map[string]string)
	bodyStart := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		closed := false
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				bodyStart = i + 1
				closed = true
				break
			}
			line := strings.TrimSpace(lines[i])
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				return Definition{}, &Error{Code: CodeInvalid, Detail: fmt.Sprintf("%s has invalid frontmatter line %q", filename, line)}
			}
			meta[strings.ToLower(strings.TrimSpace(key))] = strings.Trim(strings.TrimSpace(value), "\"'")
		}
		if !closed {
			return Definition{}, &Error{Code: CodeInvalid, Detail: filename + " has unterminated frontmatter"}
		}
	}

	name := meta["name"]
	if name == "" {
		name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}
	body := strings.Join(lines[bodyStart:], "\n")
	// A blank separator immediately after frontmatter is presentation only and
	// is not part of the prompt template.
	body = strings.TrimPrefix(body, "\n")
	if strings.TrimSpace(body) == "" {
		return Definition{}, &Error{Code: CodeInvalid, Command: name, Detail: "custom command prompt body is empty"}
	}

	return Definition{
		Name:        name,
		Aliases:     parseAliases(meta["aliases"]),
		Description: meta["description"],
		Category:    meta["category"],
		Usage:       meta["usage"],
		Source:      source,
		Action:      ActionPrompt,
		Agent:       strings.TrimSpace(meta["agent"]),
		Template:    body,
	}, nil
}

func parseAliases(raw string) []string {
	raw = strings.TrimSpace(strings.Trim(raw, "[]"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "\"'")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
