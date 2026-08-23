package app

// startTaskMetric starts one metadata-only event for the user task. Metrics are
// best-effort by design: collection failures never block the actual prompt.
func (m *Model) startTaskMetric(agent string) {
	if m.metrics == nil {
		return
	}
	eventID, err := m.metrics.StartSession(string(m.sessionID), agent, m.loadedSkillIDs)
	if err != nil {
		return
	}
	m.activeMetricID = eventID
}

func (m *Model) completeTaskMetric(status MetricStatus, errorCategory string, requestFeedback bool) {
	if m.metrics == nil || m.activeMetricID == "" {
		return
	}
	eventID := m.activeMetricID
	m.activeMetricID = ""
	// Complete updates the in-memory event before attempting persistence. Even if
	// disk persistence fails, the product task remains successful and feedback is
	// still safe to collect as metadata only.
	_ = m.metrics.Complete(eventID, status, errorCategory)
	if requestFeedback && status == MetricStatusCompleted {
		m.feedback.Open(eventID)
	}
}
