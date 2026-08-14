package components

import (
	"strings"

	"codea/tui/internal/runtime"
)

// PermissionModel renders the tool-approval modal. It consumes only the Codea
// domain runtime.ApprovalRequest and never touches the OpenCode vendor
// permission DTO or reply endpoint. The Application maps the user's key press
// to a runtime.ApprovalDecision and issues ReplyApproval itself.
type PermissionModel struct {
	Request *runtime.ApprovalRequest
}

// NewPermissionModel wraps a pending approval request for display.
func NewPermissionModel(req *runtime.ApprovalRequest) PermissionModel {
	return PermissionModel{Request: req}
}

// Visible reports whether there is a pending approval to show.
func (m PermissionModel) Visible() bool {
	return m.Request != nil
}

// View renders the approval modal, including a danger warning when the command
// matches a known dangerous fragment.
func (m PermissionModel) View() string {
	if m.Request == nil {
		return ""
	}
	tool := m.Request.Permission
	if tool == "" {
		tool = "unknown"
	}
	command := m.Request.Command
	dangerous, _ := IsDangerousCommand(command)

	lines := []string{
		"Tool approval required",
		"",
		"Tool: " + tool,
		"",
		command,
	}
	if dangerous {
		lines = append(lines, "", "⚠ Potentially dangerous command")
	}
	lines = append(lines, "", "[Y] Allow once", "[A] Always allow", "[N] Reject")
	return box(lines)
}

// box wraps lines in a bordered box padded to the widest line.
func box(lines []string) string {
	width := 0
	for _, l := range lines {
		if n := runeLen(l); n > width {
			width = n
		}
	}
	top := "┌" + strings.Repeat("─", width+2) + "┐"
	bottom := "└" + strings.Repeat("─", width+2) + "┘"
	var b strings.Builder
	b.WriteString(top)
	b.WriteString("\n")
	for _, l := range lines {
		b.WriteString("│ ")
		b.WriteString(padRight(l, width))
		b.WriteString(" │\n")
	}
	b.WriteString(bottom)
	return b.String()
}

func runeLen(s string) int {
	return len([]rune(s))
}

func padRight(s string, n int) string {
	r := []rune(s)
	for len(r) < n {
		r = append(r, ' ')
	}
	return string(r)
}
