package components

import (
	"strings"
	"testing"

	"codea/tui/internal/runtime"
)

func TestPermissionModelVisible(t *testing.T) {
	var m PermissionModel
	if m.Visible() {
		t.Fatal("zero PermissionModel should be hidden")
	}
	req := &runtime.ApprovalRequest{ID: "per_1", Permission: "shell", Command: "git status"}
	m = NewPermissionModel(req)
	if !m.Visible() {
		t.Fatal("PermissionModel with request should be visible")
	}
}

func TestPermissionModelViewDangerous(t *testing.T) {
	req := &runtime.ApprovalRequest{ID: "per_1", Permission: "shell", Command: "rm -rf ./build"}
	out := NewPermissionModel(req).View()
	for _, want := range []string{
		"Tool approval required",
		"shell",
		"rm -rf ./build",
		"Potentially dangerous command",
		"[Y] Allow once",
		"[A] Always allow",
		"[N] Reject",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
}

func TestPermissionModelViewSafeNoWarning(t *testing.T) {
	req := &runtime.ApprovalRequest{ID: "per_1", Permission: "bash", Command: "git status"}
	out := NewPermissionModel(req).View()
	if strings.Contains(out, "Potentially dangerous command") {
		t.Errorf("safe command should not show danger warning:\n%s", out)
	}
	if !strings.Contains(out, "git status") {
		t.Errorf("View should show the command:\n%s", out)
	}
}

func TestPermissionModelEmptyView(t *testing.T) {
	var m PermissionModel
	if m.View() != "" {
		t.Errorf("hidden PermissionModel View should be empty, got %q", m.View())
	}
}
