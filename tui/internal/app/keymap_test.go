package app

import "testing"

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}

func TestDefaultKeyMapBindings(t *testing.T) {
	km := DefaultKeyMap()

	cases := []struct {
		name   string
		keys   []string
		expect string
	}{
		{"Submit", km.Submit.Keys(), "enter"},
		{"Newline", km.Newline.Keys(), "alt+enter"},
		{"Newline-alt", km.Newline.Keys(), "ctrl+j"},
		{"Quit", km.Quit.Keys(), "ctrl+c"},
		{"ToggleThink", km.ToggleThink.Keys(), "ctrl+t"},
		{"ClearScreen", km.ClearScreen.Keys(), "ctrl+l"},
		{"Help", km.Help.Keys(), "?"},
	}
	for _, c := range cases {
		if !contains(c.keys, c.expect) {
			t.Errorf("%s should bind %q, got %v", c.name, c.expect, c.keys)
		}
	}
}

func TestDefaultKeyMapHelpText(t *testing.T) {
	km := DefaultKeyMap()
	if km.Submit.Help().Desc == "" {
		t.Error("Submit binding should have help text")
	}
	if km.Quit.Help().Desc == "" {
		t.Error("Quit binding should have help text")
	}
	if km.ToggleThink.Help().Desc == "" {
		t.Error("ToggleThink binding should have help text")
	}
}
