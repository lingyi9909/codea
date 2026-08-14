package components

import (
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/runtime"
)

// SessionItem is one row in the session list.
type SessionItem struct {
	ID        runtime.SessionID
	Title     string
	UpdatedAt time.Time
	Active    bool
}

// SessionModel is a pure presentation component for the session list. It owns
// only the cursor, visibility, and display of sessions; it never talks to the
// Runtime. The Application feeds it Codea-domain SessionItems and reads the
// Selected item back on Resume.
type SessionModel struct {
	Items   []SessionItem
	Cursor  int
	Visible bool
}

// Open shows the list and resets the cursor to the top.
func (m *SessionModel) Open(items []SessionItem) {
	m.Items = items
	m.Cursor = 0
	m.Visible = true
}

// Close hides the list.
func (m *SessionModel) Close() {
	m.Visible = false
}

// MoveUp moves the cursor up, clamping at the first item.
func (m *SessionModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

// MoveDown moves the cursor down, clamping at the last item.
func (m *SessionModel) MoveDown() {
	if m.Cursor < len(m.Items)-1 {
		m.Cursor++
	}
}

// Selected returns the item under the cursor, or false when empty.
func (m *SessionModel) Selected() (SessionItem, bool) {
	if len(m.Items) == 0 || m.Cursor < 0 || m.Cursor >= len(m.Items) {
		return SessionItem{}, false
	}
	return m.Items[m.Cursor], true
}

// SetActive marks the item matching id as the current session and clears the
// marker on all others.
func (m *SessionModel) SetActive(id runtime.SessionID) {
	for i := range m.Items {
		m.Items[i].Active = m.Items[i].ID == id
	}
}

// View renders the session list. The cursor line is prefixed with ">", the
// current session is tagged with "(active)".
func (m *SessionModel) View() string {
	var b strings.Builder
	b.WriteString("Sessions\n\n")
	if len(m.Items) == 0 {
		b.WriteString("  (no sessions)\n")
	} else {
		for i, item := range m.Items {
			prefix := "  "
			if i == m.Cursor {
				prefix = "> "
			}
			title := item.Title
			if title == "" {
				title = string(item.ID)
			}
			line := prefix + title
			if item.Active {
				line += " (active)"
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("↑↓ Select   Enter Resume   Esc Close"))
	return b.String()
}
