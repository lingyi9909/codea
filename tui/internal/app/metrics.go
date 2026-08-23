package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type MetricStatus string

const (
	MetricStatusCompleted MetricStatus = "completed"
	MetricStatusFailed    MetricStatus = "failed"
)

type AdoptionStatus string

const (
	AdoptionAcceptedAsIs             AdoptionStatus = "accepted_as_is"
	AdoptionAcceptedWithMinorChanges AdoptionStatus = "accepted_with_minor_changes"
	AdoptionAcceptedWithMajorChanges AdoptionStatus = "accepted_with_major_changes"
	AdoptionRejected                 AdoptionStatus = "rejected"
)

// MetricEvent is deliberately metadata-only. It has no fields for prompts,
// answers, source, diffs, tool payloads, API keys, or absolute project paths.
type MetricEvent struct {
	SchemaVersion int            `json:"schemaVersion"`
	EventID       string         `json:"eventId"`
	SessionID     string         `json:"sessionId"`
	ProjectHash   string         `json:"projectHash"`
	Agent         string         `json:"agent"`
	StartedAt     time.Time      `json:"startedAt"`
	CompletedAt   *time.Time     `json:"completedAt,omitempty"`
	DurationMs    int64          `json:"durationMs,omitempty"`
	Status        MetricStatus   `json:"status,omitempty"`
	Adoption      AdoptionStatus `json:"adoption,omitempty"`
	Rating        *int           `json:"rating,omitempty"`
	ErrorCategory *string        `json:"errorCategory,omitempty"`
	SkillsLoaded  []string       `json:"skillsLoaded,omitempty"`
}

// MetricsCollector stores one metadata-only JSON document per task. Keeping
// events separate allows feedback to update adoption without an append-only
// correction protocol while still avoiding any task content in the store.
type MetricsCollector struct {
	mu          sync.Mutex
	projectHash string
	dir         string
	events      map[string]MetricEvent
	now         func() time.Time
}

func NewMetricsCollector(projectRoot, dir string) (*MetricsCollector, error) {
	if dir != "" {
		if err := os.MkdirAll(filepath.Join(dir, "events"), 0o700); err != nil {
			return nil, fmt.Errorf("create metrics directory: %w", err)
		}
	}
	return &MetricsCollector{
		projectHash: anonymousID("project", projectRoot),
		dir:         dir,
		events:      make(map[string]MetricEvent),
		now:         time.Now,
	}, nil
}

// SetMetricsCollector enables best-effort pilot metrics for this TUI model.
// A nil collector leaves metrics disabled without changing product behavior.
func (m *Model) SetMetricsCollector(collector *MetricsCollector) {
	m.metrics = collector
}

func (c *MetricsCollector) StartSession(runtimeSessionID, agent string, skills []string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("metrics collector is nil")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	eventID, err := newMetricEventID()
	if err != nil {
		return "", err
	}
	sessionSeed := runtimeSessionID
	if sessionSeed == "" {
		// First-prompt collection can begin before OpenCode has returned the real
		// session ID. Use an event-scoped local token rather than storing an empty
		// or later back-filling a raw Runtime identifier.
		sessionSeed = eventID
	}
	event := MetricEvent{
		SchemaVersion: 1,
		EventID:       eventID,
		SessionID:     anonymousID("session", sessionSeed),
		ProjectHash:   c.projectHash,
		Agent:         agent,
		StartedAt:     c.now().UTC(),
		SkillsLoaded:  normalizedSkillIDs(skills),
	}
	c.events[eventID] = event
	if err := c.persistLocked(event); err != nil {
		delete(c.events, eventID)
		return "", err
	}
	return eventID, nil
}

func (c *MetricsCollector) Complete(eventID string, status MetricStatus, errorCategory string) error {
	if c == nil || eventID == "" {
		return nil
	}
	if status != MetricStatusCompleted && status != MetricStatusFailed {
		return fmt.Errorf("invalid metric status %q", status)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	event, ok := c.events[eventID]
	if !ok {
		return fmt.Errorf("metric event %q not found", eventID)
	}
	now := c.now().UTC()
	event.CompletedAt = &now
	event.DurationMs = max(int64(0), now.Sub(event.StartedAt).Milliseconds())
	event.Status = status
	if errorCategory != "" {
		category := errorCategory
		event.ErrorCategory = &category
	} else {
		event.ErrorCategory = nil
	}
	c.events[eventID] = event
	return c.persistLocked(event)
}

func (c *MetricsCollector) RecordFeedback(eventID string, choice FeedbackChoice) error {
	if c == nil || eventID == "" || choice == FeedbackSkip {
		return nil
	}
	adoption, ok := adoptionForFeedback(choice)
	if !ok {
		return fmt.Errorf("invalid feedback choice %q", choice)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	event, exists := c.events[eventID]
	if !exists {
		return fmt.Errorf("metric event %q not found", eventID)
	}
	event.Adoption = adoption
	c.events[eventID] = event
	return c.persistLocked(event)
}

func (c *MetricsCollector) SetRating(eventID string, rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	if c == nil || eventID == "" {
		return fmt.Errorf("metric event required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	event, ok := c.events[eventID]
	if !ok {
		return fmt.Errorf("metric event %q not found", eventID)
	}
	value := rating
	event.Rating = &value
	c.events[eventID] = event
	return c.persistLocked(event)
}

func (c *MetricsCollector) Snapshot(eventID string) (MetricEvent, bool) {
	if c == nil {
		return MetricEvent{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	event, ok := c.events[eventID]
	if !ok {
		return MetricEvent{}, false
	}
	event.SkillsLoaded = append([]string(nil), event.SkillsLoaded...)
	if event.CompletedAt != nil {
		value := *event.CompletedAt
		event.CompletedAt = &value
	}
	if event.Rating != nil {
		value := *event.Rating
		event.Rating = &value
	}
	if event.ErrorCategory != nil {
		value := *event.ErrorCategory
		event.ErrorCategory = &value
	}
	return event, true
}

func (c *MetricsCollector) persistLocked(event MetricEvent) error {
	if c.dir == "" {
		return nil
	}
	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join(c.dir, "events", event.EventID+".json")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open metric event: %w", err)
	}
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return fmt.Errorf("secure metric event: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write metric event: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close metric event: %w", err)
	}
	return nil
}

func newMetricEventID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate metric event id: %w", err)
	}
	return "evt_" + hex.EncodeToString(random[:]), nil
}

func anonymousID(prefix, value string) string {
	sum := sha256.Sum256([]byte(value))
	return prefix + "_" + hex.EncodeToString(sum[:12])
}

func normalizedSkillIDs(skills []string) []string {
	set := make(map[string]struct{}, len(skills))
	for _, name := range skills {
		if name != "" {
			set[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
