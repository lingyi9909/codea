// Package agent materializes the Codea enterprise agents
// (distribution/agents/*) into the controlled OpenCode config directory so the
// locked runtime actually loads them as first-class agents.
//
// Dependency rule: this package depends only on the standard library. It must
// never import the OpenCode vendor layer; it emits OpenCode markdown agent
// files (frontmatter + prompt body) which the runtime discovers natively.
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// OpenCodeMode is the OpenCode agent mode the enterprise agents are published
// under. "all" makes an agent selectable as a primary agent (enterprise-
// controlled mode) and delegable as a subagent.
const OpenCodeMode = "all"

// Manifest is the parsed subset of a Codea agent manifest.yaml that the
// OpenCode materialization needs. Constraints, skills and version are Codea
// metadata (already encoded in the prompt or enforced by the plugin) and are
// intentionally not carried into the OpenCode config.
type Manifest struct {
	Name        string
	DisplayName string
	// AllowTools are the tools the manifest whitelists (tools map value "allow").
	// They become the only tools the runtime permits, behind a fail-closed
	// `"*": deny` entry, so an unlisted tool is denied rather than allowed.
	AllowTools []string
	// DenyTools are tools the manifest explicitly marks deny. With the fail-closed
	// wildcard they are redundant for enforcement, but are retained so callers can
	// assert the manifest's declared intent (e.g. write/edit/bash never permitted).
	DenyTools []string
}

// ParseManifest parses the flat YAML subset of a Codea agent manifest. It only
// understands top-level `name`/`displayName` scalars and the `tools:` map; the
// remaining keys (version, mode, skills, constraints) are ignored. A manifest
// with no `name` is invalid.
func ParseManifest(body []byte) (Manifest, error) {
	var m Manifest
	allow := map[string]bool{}
	deny := map[string]bool{}
	inTools := false

	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "- ") {
			continue
		}
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		if !indented && key == "tools" {
			inTools = true
			continue
		}
		if inTools && indented && value != "" {
			switch value {
			case "allow":
				allow[key] = true
			case "deny":
				deny[key] = true
			}
			continue
		}
		if inTools && !indented {
			inTools = false
		}

		switch key {
		case "name":
			m.Name = value
		case "displayName":
			m.DisplayName = value
		}
	}

	if m.Name == "" {
		return Manifest{}, fmt.Errorf("manifest has no name")
	}
	if m.DisplayName == "" {
		m.DisplayName = m.Name
	}
	for t := range allow {
		m.AllowTools = append(m.AllowTools, t)
	}
	for t := range deny {
		m.DenyTools = append(m.DenyTools, t)
	}
	sort.Strings(m.AllowTools)
	sort.Strings(m.DenyTools)
	return m, nil
}

// Render produces the OpenCode markdown agent file for a manifest and system
// prompt. The prompt is the agent.md body verbatim; the frontmatter carries the
// description, mode and a fail-closed permission map: `"*": deny` first, then
// each whitelisted tool as `allow`. Unlisted tools therefore inherit deny rather
// than OpenCode's custom-agent default of `"*": allow`.
func Render(m Manifest, prompt string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %s\n", m.DisplayName)
	fmt.Fprintf(&b, "mode: %s\n", OpenCodeMode)
	b.WriteString("permission:\n")
	b.WriteString("  \"*\": deny\n")
	for _, t := range m.AllowTools {
		fmt.Fprintf(&b, "  %s: allow\n", t)
	}
	b.WriteString("---\n")
	if prompt != "" {
		b.WriteString(prompt)
		if !strings.HasSuffix(prompt, "\n") {
			b.WriteString("\n")
		}
	}
	return []byte(b.String())
}

// Materialize scans agentsRoot for <name>/manifest.yaml + agent.md pairs and
// writes an OpenCode markdown agent file into targetDir for each. It clears any
// previously materialized agents first so removed agents do not linger. A
// directory with a manifest.yaml that fails to parse is a hard error (a broken
// manifest must not silently produce a permissive agent); a missing agent.md is
// allowed (empty prompt).
func Materialize(agentsRoot, targetDir string) error {
	entries, err := os.ReadDir(agentsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var agents []struct {
		name string
		md   []byte
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(agentsRoot, entry.Name())
		manifestBody, err := os.ReadFile(filepath.Join(dir, "manifest.yaml"))
		if err != nil {
			if os.IsNotExist(err) {
				continue // directory without manifest.yaml is not an agent
			}
			return err
		}
		m, err := ParseManifest(manifestBody)
		if err != nil {
			return fmt.Errorf("agent %s: %w", entry.Name(), err)
		}
		prompt, _ := os.ReadFile(filepath.Join(dir, "agent.md"))
		agents = append(agents, struct {
			name string
			md   []byte
		}{name: m.Name, md: Render(m, string(prompt))})
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].name < agents[j].name })
	for _, a := range agents {
		if err := os.WriteFile(filepath.Join(targetDir, a.name+".md"), a.md, 0o644); err != nil {
			return err
		}
	}
	return nil
}
