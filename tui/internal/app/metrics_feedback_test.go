package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codea/tui/internal/skill"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMetricsCollectorPersistsOnlyAnonymousMetadata(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "secret-project")
	metricsDir := filepath.Join(t.TempDir(), "metrics")
	collector, err := NewMetricsCollector(projectRoot, metricsDir)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC)
	completed := started.Add(90 * time.Second)
	collector.now = func() time.Time { return started }

	eventID, err := collector.StartSession("runtime-session-secret", "code-reviewer", []string{"code-review"})
	if err != nil {
		t.Fatal(err)
	}
	collector.now = func() time.Time { return completed }
	if err := collector.Complete(eventID, MetricStatusCompleted, ""); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(metricsDir, "events", eventID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, forbidden := range []string{projectRoot, "runtime-session-secret", "secret-project"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("metrics leaked forbidden value %q: %s", forbidden, text)
		}
	}

	var event MetricEvent
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatal(err)
	}
	if event.SchemaVersion != 1 || event.EventID != eventID {
		t.Fatalf("unexpected identity fields: %+v", event)
	}
	if event.ProjectHash == "" || !strings.HasPrefix(event.ProjectHash, "project_") {
		t.Fatalf("project hash is not anonymized: %q", event.ProjectHash)
	}
	if event.SessionID == "" || strings.Contains(event.SessionID, "runtime-session-secret") {
		t.Fatalf("session id is not anonymized: %q", event.SessionID)
	}
	if event.Agent != "code-reviewer" || event.DurationMs != 90000 || event.Status != MetricStatusCompleted {
		t.Fatalf("unexpected event metadata: %+v", event)
	}
	if len(event.SkillsLoaded) != 1 || event.SkillsLoaded[0] != "code-review" {
		t.Fatalf("skillsLoaded=%v", event.SkillsLoaded)
	}
}

func TestMetricsFeedbackUpdatesOnlyAdoptionMetadata(t *testing.T) {
	collector, err := NewMetricsCollector(filepath.Join(t.TempDir(), "project"), filepath.Join(t.TempDir(), "metrics"))
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := collector.StartSession("session-1", "api-documentation", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.Complete(eventID, MetricStatusCompleted, ""); err != nil {
		t.Fatal(err)
	}
	if err := collector.RecordFeedback(eventID, FeedbackPartly); err != nil {
		t.Fatal(err)
	}
	event, ok := collector.Snapshot(eventID)
	if !ok {
		t.Fatal("metric event missing")
	}
	if event.Adoption != AdoptionAcceptedWithMinorChanges {
		t.Fatalf("adoption=%q", event.Adoption)
	}
}

func TestMetricsCollectorRejectsInvalidRating(t *testing.T) {
	collector, err := NewMetricsCollector(t.TempDir(), filepath.Join(t.TempDir(), "metrics"))
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := collector.StartSession("session", "general", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := collector.SetRating(eventID, 0); err == nil {
		t.Fatal("rating below 1 must be rejected")
	}
	if err := collector.SetRating(eventID, 6); err == nil {
		t.Fatal("rating above 5 must be rejected")
	}
	if err := collector.SetRating(eventID, 4); err != nil {
		t.Fatalf("valid rating rejected: %v", err)
	}
}

func TestFeedbackModelChoicesAndSkip(t *testing.T) {
	cases := []struct {
		name   string
		key    tea.KeyMsg
		want   FeedbackChoice
		handled bool
	}{
		{"yes", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, FeedbackYes, true},
		{"partly", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, FeedbackPartly, true},
		{"no", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, FeedbackNo, true},
		{"skip", tea.KeyMsg{Type: tea.KeyEsc}, FeedbackSkip, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var model FeedbackModel
			model.Open("evt_123")
			got, handled := model.HandleKey(tc.key)
			if handled != tc.handled || got != tc.want {
				t.Fatalf("got choice=%q handled=%v", got, handled)
			}
			if model.Visible() {
				t.Fatal("feedback must close after a choice or skip")
			}
		})
	}
}

func TestSkillRefreshForMetricsDoesNotOpenSkillsPage(t *testing.T) {
	m := NewModel(nil)
	m.currentPage = PageChat
	m.handleSkillListResult(listSkillsResultMsg{snapshot: skill.Snapshot{Skills: []skill.Skill{
		{Name: "api-documentation", Loaded: true},
		{Name: "code-review", Loaded: false},
	}}})
	if m.skillPanel.Visible {
		t.Fatal("background metrics skill refresh must not open the skills page")
	}
	if len(m.loadedSkillIDs) != 1 || m.loadedSkillIDs[0] != "api-documentation" {
		t.Fatalf("loadedSkillIDs=%v", m.loadedSkillIDs)
	}
}

func TestModelCompletesMetricAndRequestsSkippableFeedback(t *testing.T) {
	metricsDir := filepath.Join(t.TempDir(), "metrics")
	collector, err := NewMetricsCollector(filepath.Join(t.TempDir(), "project"), metricsDir)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(nil)
	m.SetMetricsCollector(collector)
	m.sessionID = "raw-runtime-session"
	m.loadedSkillIDs = []string{"code-review"}
	m.input = "this prompt must never enter metrics"

	_ = m.submit()
	if m.activeMetricID == "" {
		t.Fatal("submit did not start a metric event")
	}
	eventID := m.activeMetricID
	m.finishStreaming()
	if !m.feedback.Visible() || m.feedback.EventID != eventID {
		t.Fatalf("feedback not opened for completed task: %+v", m.feedback)
	}
	event, ok := collector.Snapshot(eventID)
	if !ok || event.Status != MetricStatusCompleted {
		t.Fatalf("completed metric missing: %+v ok=%v", event, ok)
	}
	data, err := os.ReadFile(filepath.Join(metricsDir, "events", eventID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "this prompt must never enter metrics") || strings.Contains(string(data), "raw-runtime-session") {
		t.Fatalf("metrics leaked task content or raw session: %s", data)
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	event, _ = collector.Snapshot(eventID)
	if event.Adoption != AdoptionAcceptedAsIs {
		t.Fatalf("adoption=%q", event.Adoption)
	}
}
