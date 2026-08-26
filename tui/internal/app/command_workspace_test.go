package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"codea/tui/internal/command"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestSlashOpensPaletteAndFiltersLive(t *testing.T) {
	m := NewModel(fakeruntime.New())

	m.handleTyping(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.commandPalette.Visible {
		t.Fatal("typing / should open command palette")
	}
	if len(m.commandPalette.Items) != 8 {
		t.Fatalf("palette items = %d, want 8", len(m.commandPalette.Items))
	}

	m.handleTyping(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m.handleTyping(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if len(m.commandPalette.Items) != 1 || m.commandPalette.Items[0].Name != "sessions" {
		t.Fatalf("filtered items = %#v", m.commandPalette.Items)
	}
}

func TestPaletteNavigationAndEscapeAreModal(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.handleTyping(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	before := m.input

	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.input != before {
		t.Fatalf("palette down leaked into input: %q -> %q", before, m.input)
	}
	if m.commandPalette.Cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.commandPalette.Cursor)
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.commandPalette.Visible {
		t.Fatal("Esc should close command palette")
	}
	if m.input != before {
		t.Fatalf("Esc changed chat input to %q", m.input)
	}
}

func TestUnknownSlashCommandNeverCreatesModelPrompt(t *testing.T) {
	fake := fakeruntime.New()
	m := NewModel(fake)
	m.input = "/definitely-unknown"

	cmd := m.submit()
	if cmd != nil {
		t.Fatal("unknown command should be handled synchronously")
	}
	if m.isStreaming {
		t.Fatal("unknown command must not enter model streaming state")
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Role != RoleInfo {
		t.Fatalf("messages = %#v, want user-visible command error", m.messages)
	}
}

func TestDirectClearCommandExecutesWithoutModelPrompt(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.messages = []ChatMessage{{Role: RoleAssistant, Content: "old", Finished: true}}
	m.input = "/clear"

	if cmd := m.submit(); cmd != nil {
		t.Fatal("clear should execute synchronously")
	}
	if len(m.messages) != 0 {
		t.Fatalf("messages = %#v, want cleared", m.messages)
	}
	if m.input != "" {
		t.Fatalf("input = %q, want empty", m.input)
	}
}

func TestPaletteEnterExecutesSelectedCommand(t *testing.T) {
	m := NewModel(fakeruntime.New())
	reg := command.NewRegistry()
	if err := reg.Register(command.Definition{Name: "clear", Description: "clear", Source: command.SourceBuiltin, Action: command.ActionClear}); err != nil {
		t.Fatal(err)
	}
	m.SetCommandRegistry(reg)
	m.input = "/cl"
	m.refreshCommandPalette()

	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("selected clear should execute synchronously")
	}
	if m.commandPalette.Visible {
		t.Fatal("palette should close after execution")
	}
	if m.input != "" {
		t.Fatalf("input = %q, want cleared", m.input)
	}
}
