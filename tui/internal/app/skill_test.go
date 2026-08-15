package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"codea/tui/internal/components"
	"codea/tui/internal/skill"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

type fakeSkillManager struct {
	snap    skill.Snapshot
	err     error
	toggles []string
}

func (f *fakeSkillManager) List(context.Context) (skill.Snapshot, error) {
	return f.snap, f.err
}

func (f *fakeSkillManager) SetEnabled(name string, enabled bool) error {
	f.toggles = append(f.toggles, name)
	return f.err
}

func TestToggleSkillsOpensPageAndFetches(t *testing.T) {
	m := NewModel(fakeruntime.New())
	fm := &fakeSkillManager{}
	m.SetSkillManager(fm)

	cmd := m.toggleSkills()

	if m.currentPage != PageSkills {
		t.Fatalf("currentPage = %v, want PageSkills", m.currentPage)
	}
	if cmd == nil {
		t.Fatal("toggleSkills should issue a ListSkillsCmd")
	}

	// Leaving the page restores chat.
	m.toggleSkills()
	if m.currentPage != PageChat {
		t.Fatalf("currentPage = %v, want PageChat after second toggle", m.currentPage)
	}
}

func TestToggleSkillsUnavailableWithoutManager(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.toggleSkills()

	if m.currentPage != PageSkills {
		t.Fatalf("currentPage = %v, want PageSkills", m.currentPage)
	}
	if !strings.Contains(m.skillNotice, "unavailable") {
		t.Fatalf("skillNotice = %q, want unavailable notice", m.skillNotice)
	}
}

func TestHandleSkillListResultPopulatesAndReportsErrors(t *testing.T) {
	m := NewModel(fakeruntime.New())
	fm := &fakeSkillManager{}
	m.SetSkillManager(fm)
	m.currentPage = PageSkills

	m.handleSkillListResult(listSkillsResultMsg{
		snapshot: skill.Snapshot{
			Skills: []skill.Skill{{Name: "git", Source: skill.SourceCodea, Installed: true, Enabled: true, Loaded: true}},
			Errors: []skill.SkillError{{Name: "broken", Stage: skill.StageDiscover, Message: "boom"}},
		},
	})

	if len(m.skillPanel.Items) != 1 || m.skillPanel.Items[0].Name != "git" {
		t.Fatalf("skillPanel.Items = %+v", m.skillPanel.Items)
	}
	if !strings.Contains(m.skillNotice, "1 skill(s) failed") {
		t.Fatalf("skillNotice = %q, want failure count", m.skillNotice)
	}
}

func TestHandleSkillListResultSurfacesError(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.SetSkillManager(&fakeSkillManager{})
	m.currentPage = PageSkills

	m.handleSkillListResult(listSkillsResultMsg{err: errors.New("down")})

	if !strings.Contains(m.skillNotice, "Failed to load skills") {
		t.Fatalf("skillNotice = %q, want load failure", m.skillNotice)
	}
}

func TestToggleSelectedSkillFlipsEnabled(t *testing.T) {
	m := NewModel(fakeruntime.New())
	fm := &fakeSkillManager{}
	m.SetSkillManager(fm)
	m.skillPanel.Open([]components.SkillItem{{Name: "git", Enabled: true}})

	cmd := m.toggleSelectedSkill()

	if cmd == nil {
		t.Fatal("toggleSelectedSkill should issue a SetSkillEnabledCmd")
	}
}

func TestHandleSkillSetResultRelistsOnSuccess(t *testing.T) {
	m := NewModel(fakeruntime.New())
	fm := &fakeSkillManager{}
	m.SetSkillManager(fm)

	cmd := m.handleSkillSetResult(setSkillResultMsg{name: "git"})

	if cmd == nil {
		t.Fatal("successful toggle should re-list")
	}
}

func TestHandleSkillSetResultSurfacesError(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.SetSkillManager(&fakeSkillManager{})
	m.currentPage = PageSkills

	m.handleSkillSetResult(setSkillResultMsg{name: "git", err: errors.New("nope")})

	if !strings.Contains(m.skillNotice, "Failed to update") {
		t.Fatalf("skillNotice = %q, want update failure", m.skillNotice)
	}
}
