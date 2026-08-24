package app

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newActiveMetricModel(t *testing.T) (*Model, *MetricsCollector, string) {
	t.Helper()
	collector, err := NewMetricsCollector(filepath.Join(t.TempDir(), "project"), filepath.Join(t.TempDir(), "metrics"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(nil)
	m.SetMetricsCollector(collector)
	m.sessionID = "runtime-session"
	m.startTaskMetric("general")
	if m.activeMetricID == "" {
		t.Fatal("expected active metric")
	}
	return &m, collector, m.activeMetricID
}

func assertFailedMetric(t *testing.T, collector *MetricsCollector, eventID, category string) {
	t.Helper()
	event, ok := collector.Snapshot(eventID)
	if !ok {
		t.Fatalf("metric %q missing", eventID)
	}
	if event.Status != MetricStatusFailed {
		t.Fatalf("status=%q want failed", event.Status)
	}
	if event.CompletedAt == nil {
		t.Fatal("failed metric must have completedAt")
	}
	if event.ErrorCategory == nil || *event.ErrorCategory != category {
		t.Fatalf("errorCategory=%v want %q", event.ErrorCategory, category)
	}
}

func TestEventStreamCloseCompletesActiveMetricAsFailed(t *testing.T) {
	m, collector, eventID := newActiveMetricModel(t)
	_, _ = m.Update(eventStreamClosedMsg{})
	if m.activeMetricID != "" {
		t.Fatalf("activeMetricID=%q want empty", m.activeMetricID)
	}
	assertFailedMetric(t, collector, eventID, "event_stream_closed")
}

func TestQuitCompletesActiveMetricAsFailed(t *testing.T) {
	m, collector, eventID := newActiveMetricModel(t)
	cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("quit key must still return tea.Quit command")
	}
	if m.activeMetricID != "" {
		t.Fatalf("activeMetricID=%q want empty", m.activeMetricID)
	}
	assertFailedMetric(t, collector, eventID, "user_quit")
}
