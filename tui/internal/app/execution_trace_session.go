package app

import "codea/tui/internal/runtime"

func (m *Model) bindActiveTraceSession(sessionID runtime.SessionID) {
	if m.activeTurnID == "" || sessionID == "" {
		return
	}
	for _, key := range []string{
		"turn:" + m.activeTurnID + ":working",
		"turn:" + m.activeTurnID + ":agent",
	} {
		idx, ok := m.executionTrace.byKey[key]
		if !ok {
			continue
		}
		m.executionTrace.entries[idx].SessionID = sessionID
	}
}
