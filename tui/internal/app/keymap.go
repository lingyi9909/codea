package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds the key bindings for the Task 7 TUI. Shift+Enter is not a
// distinct key in the bubbletea key model, so newline is bound to Alt+Enter
// and Ctrl+J.
type KeyMap struct {
	Submit      key.Binding
	Newline     key.Binding
	Quit        key.Binding
	ToggleThink key.Binding
	ClearScreen key.Binding
	Help        key.Binding
}

// DefaultKeyMap returns the default bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Newline: key.NewBinding(
			key.WithKeys("alt+enter", "ctrl+j"),
			key.WithHelp("alt+enter", "newline"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		ToggleThink: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "toggle thinking"),
		),
		ClearScreen: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
