// Command codea launches the Codea TUI against an OpenCode runtime.
//
// The composition root owns the runtime process lifecycle: by default it starts
// a supervised `opencode serve` process, waits for it to become healthy, and
// hands the resulting base URL and credentials to the vendor adapter. The app
// package itself depends only on the Codea runtime domain contract.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

	// Cold-start sync: materialize enabled Codea skills into the controlled
	// runtime config dir BEFORE the runtime starts so they are actually loaded
	// by OpenCode on first launch.
	roots := skillRoots()
	store := skill.NewFileStore(filepath.Join(cfgDir, "codea", "skills.json"))
	targetDir := filepath.Join(cfgDir, "skills")
	if err := skill.SyncEnabled(roots, store, targetDir); err != nil {
		return fmt.Errorf("sync skills: %w", err)
	}

	adapter, cleanup, err := bootstrapRuntime(cfgDir)
	if err != nil {
		return err
	}
	defer cleanup()

	projectDir, _ := os.Getwd()
	model := app.NewModel(adapter)
	model.SetSkillManager(skill.NewManager(roots, store, targetDir, projectDir, adapter))
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
func bootstrapRuntime(cfgDir string) (adapter *opencode.OpenCodeAdapter, cleanup func(), err error) {
	if baseURL := os.Getenv("OPENCODE_URL"); baseURL != "" {
		adapter = opencode.NewOpenCodeAdapter(
			baseURL,
			os.Getenv("OPENCODE_USERNAME"),
			os.Getenv("OPENCODE_PASSWORD"),
		)
		return adapter, func() {}, nil
	}

	sup := supervisor.NewSupervisor(supervisorConfig(cfgDir))
	if err := sup.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("start runtime: %w", err)
	}

	adapter = opencode.NewOpenCodeAdapter(sup.BaseURL(), sup.Username(), sup.Password())
	return adapter, func() { _ = sup.Stop() }, nil
}

// supervisorConfig builds the supervisor config. OPENCODE_BIN selects the
// OpenCode binary (default "opencode" on PATH); cfgDir is Codea's controlled
// config directory, exported to OpenCode as OPENCODE_CONFIG_DIR.
func supervisorConfig(cfgDir string) supervisor.Config {
	bin := os.Getenv("OPENCODE_BIN")
	if bin == "" {
		bin = "opencode"
	}
	projectRoot, _ := os.Getwd()
	return supervisor.Config{
		OpenCodeBin: bin,
		ConfigDir:   cfgDir,
		ProjectRoot: projectRoot,
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
