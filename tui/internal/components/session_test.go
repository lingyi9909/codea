package components

import (
	"strings"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func sampleSessions() []SessionItem {
	return []SessionItem{
		{ID: "s1", Title: "Fix OrderService bug", UpdatedAt: time.Now(), Active: true},
		{ID: "s2", Title: "Review payment module", UpdatedAt: time.Now()},
		{ID: "s3", Title: "Generate tests", UpdatedAt: time.Now()},
	}
}

func TestSessionModelOpenClose(t *testing.T) {
	var m SessionModel
	if m.Visible {
		t.Fatal("new SessionModel should start hidden")
	}
	m.Open(sampleSessions())
	if !m.Visible {
		t.Fatal("Open should set Visible")
	}
	if len(m.Items) != 3 {
		t.Fatalf("Open should set Items, got %d", len(m.Items))
	}
	m.Close()
	if m.Visible {
		t.Fatal("Close should clear Visible")
	}
}

func TestSessionModelCursorMovement(t *testing.T) {
	m := SessionModel{Items: sampleSessions(), Cursor: 0, Visible: true}
	m.MoveDown()
	if m.Cursor != 1 {
		t.Fatalf("MoveDown cursor = %d, want 1", m.Cursor)
	}
	m.MoveDown()
	m.MoveDown()
	m.MoveDown()
	if m.Cursor != 2 {
		t.Fatalf("MoveDown should clamp at last item, got %d", m.Cursor)
	}
	m.MoveUp()
	m.MoveUp()
	m.MoveUp()
	if m.Cursor != 0 {
		t.Fatalf("MoveUp should clamp at first item, got %d", m.Cursor)
	}
}

func TestSessionModelSelected(t *testing.T) {
	m := SessionModel{Items: sampleSessions(), Cursor: 1, Visible: true}
	got, ok := m.Selected()
	if !ok {
		t.Fatal("Selected should return ok for valid cursor")
	}
	if got.ID != runtime.SessionID("s2") {
		t.Fatalf("Selected ID = %q, want s2", got.ID)
	}
}

func TestSessionModelSelectedEmpty(t *testing.T) {
	var m SessionModel
	if _, ok := m.Selected(); ok {
		t.Fatal("Selected should not be ok when no items")
	}
}

func TestSessionModelSetActive(t *testing.T) {
	m := SessionModel{Items: sampleSessions()}
	m.SetActive("s2")
	if m.Items[0].Active || !m.Items[1].Active || m.Items[2].Active {
		t.Fatal("SetActive should mark exactly the matching item")
	}
}

func TestSessionModelView(t *testing.T) {
	m := SessionModel{Items: sampleSessions(), Cursor: 1, Visible: true}
	out := m.View()
	for _, want := range []string{"Sessions", "Fix OrderService bug", "Review payment module", "Generate tests", "Enter Resume", "Esc Close", "(active)"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "> Review payment module") {
		t.Errorf("View should mark cursor line with '>':\n%s", out)
	}
}

func TestSessionModelViewEmpty(t *testing.T) {
	m := SessionModel{Visible: true}
	out := m.View()
	if !strings.Contains(out, "Sessions") {
		t.Errorf("empty View should still show header:\n%s", out)
	}
}
