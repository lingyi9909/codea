package components

import (
	"fmt"
	"strings"
)

// SkillItem is one row in the skills table. Source is the display form of a
// Codea skill.SkillSource string; the component never interprets it.
type SkillItem struct {
	Name        string
	Description string
	Source      string
	Installed   bool
	Enabled     bool
	Loaded      bool
	LoadError   string
}

// SkillModel is a pure presentation component for the skills page. It owns only
// the cursor, visibility, and display of skills; it never talks to the Runtime
// or the Skill Manager. The Application feeds it SkillItems and reads the
// Selected item back to toggle it.
type SkillModel struct {
	Items   []SkillItem
	Cursor  int
	Visible bool
}

// Open shows the list and resets the cursor to the top.
func (m *SkillModel) Open(items []SkillItem) {
	m.Items = items
	m.Cursor = 0
	m.Visible = true
}

// Close hides the list.
func (m *SkillModel) Close() {
	m.Visible = false
}

// MoveUp moves the cursor up, clamping at the first item.
func (m *SkillModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

// MoveDown moves the cursor down, clamping at the last item.
func (m *SkillModel) MoveDown() {
	if m.Cursor < len(m.Items)-1 {
		m.Cursor++
	}
}

// Selected returns the item under the cursor, or false when empty.
func (m *SkillModel) Selected() (SkillItem, bool) {
	if len(m.Items) == 0 || m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return SkillItem{}, false
	}
	return m.Items[m.Cursor], true
}

// View renders the skills table. The cursor line is prefixed with ">". Each row
// shows the skill name, source, and its installed/enabled/loaded flags; a
// non-empty LoadError is appended.
func (m *SkillModel) View() string {
	var b strings.Builder
	b.WriteString("Skills\n\n")
	if len(m.Items) == 0 {
		b.WriteString("  (no skills)\n")
	} else {
		for i, item := range m.Items {
			prefix := "  "
			if i == m.Cursor {
				prefix = "> "
			}
			b.WriteString(prefix + renderSkillRow(item))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString("↑↓ Select   Enter Toggle   r Refresh   Esc Close")
	return b.String()
}

func renderSkillRow(item SkillItem) string {
	row := fmt.Sprintf("%s  [%s]  installed=%s enabled=%s loaded=%s",
		item.Name, item.Source, flag(item.Installed), flag(item.Enabled), flag(item.Loaded))
	if item.LoadError != "" {
		row += "  ! " + item.LoadError
	}
	return row
}

func flag(v bool) string {
	if v {
		return "✓"
	}
	return "✗"
}
