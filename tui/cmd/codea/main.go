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

	"codea/tui/internal/app"
	"codea/tui/internal/opencode"
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
	adapter, cleanup, err := bootstrapRuntime()
	if err != nil {
		return err
	}
	defer cleanup()

	model := app.NewModel(adapter)
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
func bootstrapRuntime() (adapter *opencode.OpenCodeAdapter, cleanup func(), err error) {
	if baseURL := os.Getenv("OPENCODE_URL"); baseURL != "" {
		adapter = opencode.NewOpenCodeAdapter(
			baseURL,
			os.Getenv("OPENCODE_USERNAME"),
			os.Getenv("OPENCODE_PASSWORD"),
		)
		return adapter, func() {}, nil
	}

	sup := supervisor.NewSupervisor(supervisorConfig())
	if err := sup.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("start runtime: %w", err)
	}

	adapter = opencode.NewOpenCodeAdapter(sup.BaseURL(), sup.Username(), sup.Password())
	return adapter, func() { _ = sup.Stop() }, nil
}

// supervisorConfig builds the supervisor config from environment. OPENCODE_BIN
// selects the OpenCode binary (default "opencode" on PATH); OPENCODE_CONFIG_DIR
// selects the config directory; the project root is the current directory.
func supervisorConfig() supervisor.Config {
	bin := os.Getenv("OPENCODE_BIN")
	if bin == "" {
		bin = "opencode"
	}
	projectRoot, _ := os.Getwd()
	return supervisor.Config{
		OpenCodeBin: bin,
		ConfigDir:   os.Getenv("OPENCODE_CONFIG_DIR"),
		ProjectRoot: projectRoot,
	}
}
