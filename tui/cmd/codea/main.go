// Command codea launches the Codea TUI against an OpenCode runtime.
//
// The composition root owns the runtime process lifecycle: by default it starts
// a supervised `opencode serve` process, waits for it to become healthy, and
// hands the resulting base URL and credentials to the vendor adapter. The app
// package itself depends only on the Codea runtime domain contract.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"codea/tui/internal/agent"
	"codea/tui/internal/app"
	"codea/tui/internal/opencode"
	"codea/tui/internal/skill"
	"codea/tui/internal/supervisor"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run bootstraps the runtime, runs the TUI, and stops the runtime on exit.
func run() error {
	cfgDir := codeaConfigDir()

	mode, err := skill.ResolveSkillMode(os.Getenv("CODEA_SKILL_MODE"))
	if err != nil {
		return err
	}
	policy := skill.SkillPolicy{
		Mode:     mode,
		Approved: skill.ParseApprovedSkills(os.Getenv("CODEA_APPROVED_SKILLS")),
	}

	// Cold-start sync: materialize the mode-policy-approved enabled Codea skills
	// into the controlled runtime config dir BEFORE the runtime starts so they
	// are actually loaded by OpenCode on first launch.
	roots := skillRoots()
	store := skill.NewFileStore(filepath.Join(cfgDir, "codea", "skills.json"))
	targetDir := filepath.Join(cfgDir, "skills")
	if err := skill.SyncEnabled(roots, store, targetDir, policy); err != nil {
		return fmt.Errorf("sync skills: %w", err)
	}

	// Register the enterprise plugin bundle in Codea-owned opencode.json BEFORE the
	// runtime starts, so OpenCode loads the plugin (and its 7 custom tools + DLP)
	// on first launch. A missing bundle (not yet built) degrades to General mode.
	if err := writePluginConfig(cfgDir); err != nil {
		return fmt.Errorf("write plugin config: %w", err)
	}

	// Materialize the enterprise agents (code-reviewer, unit-test-generator, …)
	// into the controlled config dir BEFORE the runtime starts, so they are
	// loaded as real first-class agents (appear in /agent) with their deny
	// permissions enforced server-side, not just as prompt instructions.
	if err := agent.Materialize(agentRoot(), filepath.Join(cfgDir, "agents")); err != nil {
		return fmt.Errorf("materialize agents: %w", err)
	}

	adapter, cleanup, err := bootstrapRuntime(cfgDir, mode)
	if err != nil {
		return err
	}
	defer cleanup()

	projectDir, _ := os.Getwd()
	model := app.NewModel(adapter)
	model.SetSkillManager(skill.NewManager(roots, store, targetDir, projectDir, adapter, policy))
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}

// bootstrapRuntime resolves the runtime connection. The product default starts
// a supervised `opencode serve` process and returns an adapter wired to the
// supervisor's auto-generated base URL and credentials, with a cleanup func
// that stops the process. When OPENCODE_URL is set (dev/test override), the
// process is assumed to be managed externally and cleanup is a no-op.
func bootstrapRuntime(cfgDir string, mode skill.SkillMode) (adapter *opencode.OpenCodeAdapter, cleanup func(), err error) {
	if baseURL := os.Getenv("OPENCODE_URL"); baseURL != "" {
		adapter = opencode.NewOpenCodeAdapter(
			baseURL,
			os.Getenv("OPENCODE_USERNAME"),
			os.Getenv("OPENCODE_PASSWORD"),
		)
		return adapter, func() {}, nil
	}

	sup := supervisor.NewSupervisor(supervisorConfig(cfgDir, mode))
	if err := sup.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("start runtime: %w", err)
	}

	adapter = opencode.NewOpenCodeAdapter(sup.BaseURL(), sup.Username(), sup.Password())
	return adapter, func() { _ = sup.Stop() }, nil
}

// supervisorConfig builds the supervisor config. OPENCODE_BIN selects the
// OpenCode binary (default "opencode" on PATH); cfgDir is Codea's controlled
// config directory, exported to OpenCode as OPENCODE_CONFIG_DIR.
func supervisorConfig(cfgDir string, mode skill.SkillMode) supervisor.Config {
	bin := os.Getenv("OPENCODE_BIN")
	if bin == "" {
		bin = "opencode"
	}
	projectRoot, _ := os.Getwd()
	return supervisor.Config{
		OpenCodeBin:     bin,
		ConfigDir:       cfgDir,
		ProjectRoot:     projectRoot,
		CodeaSkillsOnly: mode == skill.SkillModeStrict,
	}
}

// codeaConfigDir returns the Codea-owned controlled config directory. It is a
// dedicated location (never OpenCode's native ~/.config/opencode), so the skill
// sync's RemoveAll can only ever affect Codea's own files and never delete a
// user's existing OpenCode skills.
func codeaConfigDir() string {
	if d := os.Getenv("CODEA_RUNTIME_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codea", "runtime-config")
}

// agentRoot returns the filesystem root that holds the Codea enterprise agents
// (each a <name>/manifest.yaml + agent.md directory). CODEA_AGENTS_DIR overrides
// the default (the distribution agents directory relative to the launch dir).
func agentRoot() string {
	if d := os.Getenv("CODEA_AGENTS_DIR"); d != "" {
		return d
	}
	projectDir, _ := os.Getwd()
	return filepath.Join(projectDir, "..", "distribution", "agents")
}

// pluginBundlePath returns the path to the self-contained enterprise plugin
// bundle. CODEA_PLUGIN_BUNDLE overrides the default (the distribution build
// output relative to the launch directory).
func pluginBundlePath() string {
	if p := os.Getenv("CODEA_PLUGIN_BUNDLE"); p != "" {
		return p
	}
	projectDir, _ := os.Getwd()
	return filepath.Join(projectDir, "..", "distribution", "plugins", "dist", "index.js")
}

// writePluginConfig materializes a Codea-owned opencode.json into the controlled
// config dir, registering the enterprise plugin bundle via a file:// URL. This
// is what makes the plugin (and its 7 custom tools + DLP) actually load in the
// supervised runtime. It never touches OpenCode's native ~/.config/opencode.
func writePluginConfig(cfgDir string) error {
	bundle := pluginBundlePath()
	if bundle == "" {
		return nil
	}
	// Require the bundle to exist so a stale/missing override degrades to
	// General mode rather than registering a dead plugin URL.
	if _, err := os.Stat(bundle); err != nil {
		return nil
	}
	abs, err := filepath.Abs(bundle)
	if err != nil {
		return err
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	cfg := map[string]any{
		"plugin": []string{u.String()},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfgDir, "opencode.json"), data, 0o644)
}

// skillRoots returns the filesystem roots Codea scans to display skills. Codea
// skills live in the distribution; project/user roots are read-only and are
// scanned only so they can be shown, never managed.
func skillRoots() []skill.Root {
	home, _ := os.UserHomeDir()
	projectDir, _ := os.Getwd()

	codeaSkills := os.Getenv("CODEA_SKILLS_DIR")
	if codeaSkills == "" {
		codeaSkills = filepath.Join(projectDir, "..", "distribution", "skills")
	}

	roots := []skill.Root{
		{Dir: codeaSkills, Source: skill.SourceCodea},
		{Dir: filepath.Join(projectDir, ".opencode", "skills"), Source: skill.SourceProject},
		{Dir: filepath.Join(projectDir, ".agents", "skills"), Source: skill.SourceProject},
	}
	if home != "" {
		roots = append(roots,
			skill.Root{Dir: filepath.Join(home, ".claude", "skills"), Source: skill.SourceUser},
			// The user's default OpenCode skills dir is always a read-only User
			// root now that Codea's controlled dir is isolated under ~/.codea.
			skill.Root{Dir: filepath.Join(home, ".config", "opencode", "skills"), Source: skill.SourceUser},
		)
	}
	return roots
}
