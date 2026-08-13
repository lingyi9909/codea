package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTokyoNightColorConstants(t *testing.T) {
	cases := []struct {
		name string
		got  lipgloss.Color
		want string
	}{
		{"Primary", Primary, "#c0caf5"},
		{"Secondary", Secondary, "#a9b1d6"},
		{"Muted", Muted, "#565f89"},
		{"Accent", Accent, "#e0af68"},
		{"Success", Success, "#9ece6a"},
		{"Error", Error, "#f7768e"},
		{"Border", Border, "#292e42"},
		{"Background", Background, "#1a1b26"},
	}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("%s = %q, want %q", c.name, string(c.got), c.want)
		}
	}
}

func TestStyleFunctionsForeground(t *testing.T) {
	cases := []struct {
		name  string
		style lipgloss.Style
		want  string
	}{
		{"ChatStyle", ChatStyle(), "#c0caf5"},
		{"MutedStyle", MutedStyle(), "#565f89"},
		{"AccentStyle", AccentStyle(), "#e0af68"},
		{"SuccessStyle", SuccessStyle(), "#9ece6a"},
		{"ErrorStyle", ErrorStyle(), "#f7768e"},
	}
	for _, c := range cases {
		fg, ok := c.style.GetForeground().(lipgloss.Color)
		if !ok {
			t.Errorf("%s: foreground is %T, want lipgloss.Color", c.name, c.style.GetForeground())
			continue
		}
		if string(fg) != c.want {
			t.Errorf("%s foreground = %q, want %q", c.name, string(fg), c.want)
		}
	}
}

func TestMutedStyleIsItalic(t *testing.T) {
	if !MutedStyle().GetItalic() {
		t.Error("MutedStyle should be italic")
	}
}
