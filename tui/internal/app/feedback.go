package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type FeedbackChoice string

const (
	FeedbackYes    FeedbackChoice = "yes"
	FeedbackPartly FeedbackChoice = "partly"
	FeedbackNo     FeedbackChoice = "no"
	FeedbackSkip   FeedbackChoice = "skip"
)

// FeedbackModel is intentionally lightweight: a completed task can be rated
// Yes/Partly/No or skipped with Esc. It stores only the metric event ID and no
// task content.
type FeedbackModel struct {
	EventID string
	visible bool
}

func (m *FeedbackModel) Open(eventID string) {
	if eventID == "" {
		return
	}
	m.EventID = eventID
	m.visible = true
}

func (m *FeedbackModel) Close() {
	m.visible = false
}

func (m FeedbackModel) Visible() bool { return m.visible }

// HandleKey returns a feedback choice only for the four supported actions. A
// visible feedback prompt swallows unrelated keys in the application layer so
// they cannot accidentally become a new chat prompt before feedback is closed.
func (m *FeedbackModel) HandleKey(msg tea.KeyMsg) (FeedbackChoice, bool) {
	if !m.visible {
		return "", false
	}
	choice := FeedbackChoice("")
	switch msg.Type {
	case tea.KeyEsc:
		choice = FeedbackSkip
	case tea.KeyRunes:
		if len(msg.Runes) == 1 {
			switch strings.ToLower(string(msg.Runes[0])) {
			case "y":
				choice = FeedbackYes
			case "p":
				choice = FeedbackPartly
			case "n":
				choice = FeedbackNo
			}
		}
	}
	if choice == "" {
		return "", false
	}
	m.visible = false
	return choice, true
}

func (m FeedbackModel) View() string {
	if !m.visible {
		return ""
	}
	return "Was this result useful?  [Y] Yes   [P] Partly   [N] No   [Esc] Skip"
}

func adoptionForFeedback(choice FeedbackChoice) (AdoptionStatus, bool) {
	switch choice {
	case FeedbackYes:
		return AdoptionAcceptedAsIs, true
	case FeedbackPartly:
		return AdoptionAcceptedWithMinorChanges, true
	case FeedbackNo:
		return AdoptionRejected, true
	default:
		return "", false
	}
}
