package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestNewModelDefaults(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)

	if m.currentPage != PageChat {
		t.Errorf("currentPage = %v, want PageChat", m.currentPage)
	}
	if m.runtimeStatus != runtime.RuntimeStopped {
		t.Errorf("runtimeStatus = %q, want stopped", m.runtimeStatus)
	}
	if m.runtimeClient != client {
		t.Error("runtimeClient not set to the provided client")
	}
	if len(m.messages) != 0 {
		t.Errorf("messages = %d items, want 0", len(m.messages))
	}
	if m.proc == nil {
		t.Error("reasoning processor is nil")
	}
	if len(m.tools) != 0 {
		t.Errorf("tools = %d items, want 0", len(m.tools))
	}
	if m.keys.Submit.Keys() == nil {
		t.Error("keymap not initialized")
	}
}

func TestRoleConstants(t *testing.T) {
	cases := []struct {
		got  Role
		want string
	}{
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleInfo, "info"},
	}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("role %q, want %q", c.got, c.want)
		}
	}
}

func TestToolStatusConstants(t *testing.T) {
	cases := []struct {
		got  ToolStatus
		want string
	}{
		{ToolRunning, "running"},
		{ToolSuccess, "success"},
		{ToolFailed, "failed"},
	}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("tool status %q, want %q", c.got, c.want)
		}
	}
}
