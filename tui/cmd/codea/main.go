// Command codea launches the Codea TUI against an OpenCode runtime.
//
// The composition root wires the vendor adapter into the app model; the app
// package itself depends only on the Codea runtime domain contract.
package main

import (
	"fmt"
	"os"

	"codea/tui/internal/app"
	"codea/tui/internal/opencode"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	baseURL := os.Getenv("OPENCODE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:49321"
	}
	username := os.Getenv("OPENCODE_USERNAME")
	password := os.Getenv("OPENCODE_PASSWORD")

	adapter := opencode.NewOpenCodeAdapter(baseURL, username, password)
	model := app.NewModel(adapter)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
