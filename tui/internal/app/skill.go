package app

import (
	"context"
	"fmt"
	"sort"

	"codea/tui/internal/components"
	"codea/tui/internal/skill"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// skillManager is the subset of skill.Manager the TUI drives. It is an
// interface so app tests can substitute a fake without touching the real
// manager or the OpenCode vendor layer.
type skillManager interface {
	List(ctx context.Context) (skill.Snapshot, error)
	SetEnabled(name string, enabled bool) error
}

// listSkillsResultMsg carries the result of an async skill list fetch.
type listSkillsResultMsg struct {
	snapshot skill.Snapshot
	err      error
}

// setSkillResultMsg carries the result of an async enable/disable toggle.
type setSkillResultMsg struct {
	name string
	err  error
}

// ListSkillsCmd fetches the skill snapshot without blocking the event loop.
func ListSkillsCmd(mgr skillManager) tea.Cmd {
	return func() tea.Msg {
		snap, err := mgr.List(context.Background())
		return listSkillsResultMsg{snapshot: snap, err: err}
	}
}

// SetSkillEnabledCmd persists an enable/disable toggle without blocking the
// event loop.
func SetSkillEnabledCmd(mgr skillManager, name string, enabled bool) tea.Cmd {
	return func() tea.Msg {
		err := mgr.SetEnabled(name, enabled)
		return setSkillResultMsg{name: name, err: err}
	}
}

// toggleSkills switches between the chat page and the skills page. Entering the
// skills page triggers an async list fetch; leaving it restores the chat page.
func (m *Model) toggleSkills() tea.Cmd {
	if m.currentPage == PageSkills {
		m.currentPage = PageChat
		m.skillPanel.Close()
		m.skillNotice = ""
		m.markDirty()
		return nil
	}
	m.currentPage = PageSkills
	m.skillNotice = ""
	m.markDirty()
	if m.skills == nil {
		m.skillNotice = "Skill manager unavailable."
		return nil
	}
	return ListSkillsCmd(m.skills)
}

// handleSkillKey routes keys while the skills page is active.
func (m *Model) handleSkillKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.skillPanel.MoveUp()
		m.skillNotice = ""
		return nil
	case key.Matches(msg, m.keys.Down):
		m.skillPanel.MoveDown()
		m.skillNotice = ""
		return nil
	case key.Matches(msg, m.keys.Submit):
		return m.toggleSelectedSkill()
	case key.Matches(msg, m.keys.Refresh):
		m.skillNotice = ""
		return ListSkillsCmd(m.skills)
	case key.Matches(msg, m.keys.Esc), key.Matches(msg, m.keys.Skills):
		m.currentPage = PageChat
		m.skillPanel.Close()
		m.skillNotice = ""
		return nil
	}
	return nil
}

// toggleSelectedSkill flips the enabled state of the skill under the cursor and
// re-fetches the snapshot so the UI reflects the persisted + runtime-loaded
// state rather than an optimistic local flip. Only Codea skills are toggleable;
// project/user/runtime skills are read-only.
func (m *Model) toggleSelectedSkill() tea.Cmd {
	item, ok := m.skillPanel.Selected()
	if !ok {
		return nil
	}
	if item.Source != string(skill.SourceCodea) {
		m.skillNotice = "Only Codea skills can be enabled or disabled."
		return nil
	}
	m.skillNotice = ""
	return SetSkillEnabledCmd(m.skills, item.Name, !item.Enabled)
}

// handleSkillListResult always refreshes the metadata-only loaded Skill IDs for
// Task 20 metrics. A background refresh while on Chat must not open the Skills
// page or surface its presentation state.
func (m *Model) handleSkillListResult(msg listSkillsResultMsg) tea.Cmd {
	if msg.err != nil {
		if m.currentPage == PageSkills {
			m.skillNotice = "Failed to load skills: " + msg.err.Error()
			m.markDirty()
		}
		return nil
	}
	m.loadedSkillIDs = loadedSkillNames(msg.snapshot.Skills)
	if m.currentPage != PageSkills {
		return nil
	}
	m.skillPanel.Open(skillItems(msg.snapshot.Skills))
	if len(msg.snapshot.Errors) > 0 {
		m.skillNotice = fmt.Sprintf("%d skill(s) failed to load", len(msg.snapshot.Errors))
	} else {
		m.skillNotice = ""
	}
	m.markDirty()
	return nil
}

// handleSkillSetResult surfaces a toggle failure, or re-lists on success.
func (m *Model) handleSkillSetResult(msg setSkillResultMsg) tea.Cmd {
	if msg.err != nil {
		m.skillNotice = fmt.Sprintf("Failed to update %q: %v", msg.name, msg.err)
		m.markDirty()
		return nil
	}
	m.skillNotice = ""
	return ListSkillsCmd(m.skills)
}

// skillItems maps the Codea skill domain model into presentation items.
func skillItems(skills []skill.Skill) []components.SkillItem {
	items := make([]components.SkillItem, len(skills))
	for i, s := range skills {
		items[i] = components.SkillItem{
			Name:        s.Name,
			Description: s.Description,
			Source:      string(s.Source),
			Installed:   s.Installed,
			Enabled:     s.Enabled,
			Loaded:      s.Loaded,
			LoadError:   s.LoadError,
		}
	}
	return items
}

func loadedSkillNames(skills []skill.Skill) []string {
	out := make([]string, 0, len(skills))
	for _, s := range skills {
		if s.Loaded && s.Name != "" {
			out = append(out, s.Name)
		}
	}
	sort.Strings(out)
	return out
}
