package components

import (
	"strings"
	"testing"
)

func TestSkillModelSelectedAndClamp(t *testing.T) {
	m := SkillModel{}
	m.Open([]SkillItem{{Name: "git"}, {Name: "review"}})

	item, ok := m.Selected()
	if !ok || item.Name != "git" {
		t.Fatalf("Selected = %+v, %v; want git", item, ok)
	}

	m.MoveDown()
	item, ok = m.Selected()
	if !ok || item.Name != "review" {
		t.Fatalf("Selected after MoveDown = %+v, %v; want review", item, ok)
	}

	m.MoveDown() // clamped at last
	item, _ = m.Selected()
	if item.Name != "review" {
		t.Fatalf("MoveDown should clamp at last, got %q", item.Name)
	}

	m.MoveUp()
	m.MoveUp() // clamped at first
	item, _ = m.Selected()
	if item.Name != "git" {
		t.Fatalf("MoveUp should clamp at first, got %q", item.Name)
	}
}

func TestSkillModelSelectedEmpty(t *testing.T) {
	m := SkillModel{}
	m.Open(nil)
	if _, ok := m.Selected(); ok {
		t.Fatal("Selected should report false on empty list")
	}
}

func TestSkillModelViewShowsStatusAndError(t *testing.T) {
	m := SkillModel{}
	m.Open([]SkillItem{
		{Name: "git", Source: "codea", Installed: true, Enabled: false, Loaded: false},
		{Name: "review", Source: "runtime", Installed: true, Enabled: true, Loaded: true, LoadError: "boom"},
	})
	out := m.View()
	if !strings.Contains(out, "enabled=✗") {
		t.Errorf("View should show disabled flag: %s", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("View should show LoadError: %s", out)
	}
	if !strings.Contains(out, "> git") {
		t.Errorf("View should mark cursor row: %s", out)
	}
}
