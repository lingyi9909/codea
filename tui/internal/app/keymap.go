package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds the key bindings for the TUI. Shift+Enter is not a distinct key
// in the bubbletea key model, so newline is bound to Alt+Enter and Ctrl+J.
type KeyMap struct {
	Submit      key.Binding
	Newline     key.Binding
	Quit        key.Binding
	ToggleThink key.Binding
	ClearScreen key.Binding
	Help        key.Binding

	Sessions    key.Binding
	Skills      key.Binding
	Refresh     key.Binding
	Up          key.Binding
	Down        key.Binding
	Esc         key.Binding
	AllowOnce   key.Binding
	AllowAlways key.Binding
	Reject      key.Binding
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
		Sessions: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "sessions"),
		),
		Skills: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("ctrl+k", "skills"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "select"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "select"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		AllowOnce: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "allow once"),
		),
		AllowAlways: key.NewBinding(
			key.WithKeys("a", "r"),
			key.WithHelp("a", "always allow"),
		),
		Reject: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "reject"),
		),
	}
}
