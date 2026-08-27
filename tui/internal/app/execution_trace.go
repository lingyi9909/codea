package app

import (
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/runtime"
)

// TraceCategory is a Codea-owned semantic execution category. It is deliberately
// independent from Runtime vendor event names and UI layout choices.
type TraceCategory string

const (
	TraceUser      TraceCategory = "user"
	TraceWorking   TraceCategory = "working"
	TraceAgent     TraceCategory = "agent"
	TraceTool      TraceCategory = "tool"
	TraceSkill     TraceCategory = "skill"
	TracePlugin    TraceCategory = "plugin"
	TraceApproval  TraceCategory = "approval"
	TraceAssistant TraceCategory = "assistant"
	TraceSubagent  TraceCategory = "subagent"
	TraceRuntime   TraceCategory = "runtime"
)

// TraceStatus describes the truthful lifecycle of one semantic invocation.
type TraceStatus string

const (
	TraceQueued  TraceStatus = "queued"
	TraceRunning TraceStatus = "running"
	TraceWaiting TraceStatus = "waiting"
	TraceSuccess TraceStatus = "success"
	TraceFailed  TraceStatus = "failed"
	TraceDenied  TraceStatus = "denied"
	TraceUnknown TraceStatus = "unknown"
)

// ExecutionTraceEntry is Codea's semantic execution trace contract. Rendering
// modes derive from these entries but never mutate their truth.
type ExecutionTraceEntry struct {
	Category      TraceCategory
	Title         string
	Detail        string
	Status        TraceStatus
	InvocationKey string
	StartedAt     time.Time
	FinishedAt    time.Time
	Duration      time.Duration
	SessionID     runtime.SessionID
	TurnID        string
	Metadata      map[string]string
}

type executionTrace struct {
	entries  []ExecutionTraceEntry
	byKey    map[string]int
	localSeq uint64
}

func newExecutionTrace() executionTrace {
	return executionTrace{
		entries: make([]ExecutionTraceEntry, 0),
		byKey:   make(map[string]int),
	}
}

func (t *executionTrace) Reset() {
	t.entries = make([]ExecutionTraceEntry, 0)
	t.byKey = make(map[string]int)
	t.localSeq = 0
}

func (t *executionTrace) Entries() []ExecutionTraceEntry {
	out := make([]ExecutionTraceEntry, len(t.entries))
	for i, entry := range t.entries {
		out[i] = entry
		if entry.Metadata != nil {
			out[i].Metadata = cloneTraceMetadata(entry.Metadata)
		}
	}
	return out
}

func (t *executionTrace) Entry(key string) (ExecutionTraceEntry, bool) {
	idx, ok := t.byKey[key]
	if !ok || idx < 0 || idx >= len(t.entries) {
		return ExecutionTraceEntry{}, false
	}
	entry := t.entries[idx]
	if entry.Metadata != nil {
		entry.Metadata = cloneTraceMetadata(entry.Metadata)
	}
	return entry, true
}

func (t *executionTrace) localKey(category TraceCategory) string {
	t.localSeq++
	return fmt.Sprintf("%s:local-%d", category, t.localSeq)
}

func (t *executionTrace) upsert(entry ExecutionTraceEntry) string {
	if strings.TrimSpace(entry.InvocationKey) == "" {
		entry.InvocationKey = t.localKey(entry.Category)
		if entry.Metadata == nil {
			entry.Metadata = map[string]string{}
		}
		entry.Metadata["identity"] = "unavailable"
	}
	if idx, ok := t.byKey[entry.InvocationKey]; ok {
		current := &t.entries[idx]
		if entry.Category != "" {
			current.Category = entry.Category
		}
		if strings.TrimSpace(entry.Title) != "" {
			current.Title = entry.Title
		}
		if strings.TrimSpace(entry.Detail) != "" {
			current.Detail = entry.Detail
		}
		if entry.Status != "" {
			current.Status = entry.Status
		}
		if current.StartedAt.IsZero() && !entry.StartedAt.IsZero() {
			current.StartedAt = entry.StartedAt
		}
		if !entry.FinishedAt.IsZero() {
			current.FinishedAt = entry.FinishedAt
			if !current.StartedAt.IsZero() && !current.FinishedAt.Before(current.StartedAt) {
				current.Duration = current.FinishedAt.Sub(current.StartedAt)
			}
		}
		if entry.SessionID != "" {
			current.SessionID = entry.SessionID
		}
		if entry.TurnID != "" {
			current.TurnID = entry.TurnID
		}
		if entry.Metadata != nil {
			if current.Metadata == nil {
				current.Metadata = map[string]string{}
			}
			for k, v := range entry.Metadata {
				current.Metadata[k] = v
			}
		}
		return entry.InvocationKey
	}
	if entry.Metadata != nil {
		entry.Metadata = cloneTraceMetadata(entry.Metadata)
	}
	if entry.Status == "" {
		entry.Status = TraceUnknown
	}
	if entry.StartedAt.IsZero() {
		entry.StartedAt = time.Now()
	}
	idx := len(t.entries)
	t.entries = append(t.entries, entry)
	t.byKey[entry.InvocationKey] = idx
	return entry.InvocationKey
}

func (t *executionTrace) setStatus(key string, status TraceStatus, finished bool) bool {
	idx, ok := t.byKey[key]
	if !ok {
		return false
	}
	entry := &t.entries[idx]
	entry.Status = status
	if finished {
		entry.FinishedAt = time.Now()
		if !entry.StartedAt.IsZero() && !entry.FinishedAt.Before(entry.StartedAt) {
			entry.Duration = entry.FinishedAt.Sub(entry.StartedAt)
		}
	} else {
		entry.FinishedAt = time.Time{}
		entry.Duration = 0
	}
	return true
}

func cloneTraceMetadata(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (m *Model) beginPromptTrace(req runtime.PromptRequest) {
	now := time.Now()
	m.activeTurnID = req.MessageID
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceWorking,
		Title:         "Working",
		Status:        TraceRunning,
		InvocationKey: "turn:" + req.MessageID + ":working",
		StartedAt:     now,
		SessionID:     m.sessionID,
		TurnID:        req.MessageID,
	})
	agent := strings.TrimSpace(req.Agent)
	if agent == "" {
		agent = "general"
	}
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceAgent,
		Title:         agent,
		Status:        TraceRunning,
		InvocationKey: "turn:" + req.MessageID + ":agent",
		StartedAt:     now,
		SessionID:     m.sessionID,
		TurnID:        req.MessageID,
	})
}

func (m *Model) traceRuntimeEvent(ev runtime.Event) {
	m.traceStructuredMetadata(ev)

	switch ev.Type {
	case eventTypeToolCalled:
		m.traceTool(ev, TraceRunning, false)
	case eventTypeToolSuccess:
		m.traceTool(ev, TraceSuccess, true)
	case eventTypeToolFailed:
		m.traceTool(ev, TraceFailed, true)
	case eventTypeApprovalRequested:
		m.traceApprovalRequested(ev)
	case eventTypeStepFinished:
		m.finishActiveTurnTrace(TraceSuccess)
	case eventTypeSessionError, eventTypeRuntimeError:
		m.traceRuntimeFailure(ev)
		m.finishActiveTurnTrace(TraceFailed)
	}
}

func (m *Model) traceTool(ev runtime.Event, status TraceStatus, finished bool) {
	if ev.Tool == nil {
		return
	}
	callID := strings.TrimSpace(ev.Tool.CallID)
	key := ""
	if callID != "" {
		key = "tool:" + callID
	} else if status != TraceRunning {
		// Without an invocation identity a terminal event cannot safely be
		// correlated to any prior tool with the same display name.
		key = m.executionTrace.localKey(TraceTool)
	}
	entry := ExecutionTraceEntry{
		Category:      TraceTool,
		Title:         safeTraceText(ev.Tool.Name),
		Detail:        safeTraceDetail(ev),
		Status:        status,
		InvocationKey: key,
		SessionID:     runtime.SessionID(ev.SessionID),
		TurnID:        m.activeTurnID,
	}
	if entry.Title == "" {
		entry.Title = "unavailable"
	}
	if key == "" {
		entry.Metadata = map[string]string{"identity": "unavailable"}
	}
	key = m.executionTrace.upsert(entry)
	if finished {
		m.executionTrace.setStatus(key, status, true)
	}
}

func (m *Model) traceApprovalRequested(ev runtime.Event) {
	if ev.Approval == nil {
		return
	}
	approvalID := strings.TrimSpace(ev.Approval.ID)
	key := ""
	if approvalID != "" {
		key = "approval:" + approvalID
	}
	title := safeTraceText(ev.Approval.Permission)
	if title == "" {
		title = "unavailable"
	}
	detail := safeTraceText(ev.Approval.Command)
	if ev.RawSensitivity == runtime.SensitivitySensitive {
		detail = "[sensitive command hidden]"
	}
	key = m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceApproval,
		Title:         title,
		Detail:        detail,
		Status:        TraceWaiting,
		InvocationKey: key,
		SessionID:     runtime.SessionID(ev.SessionID),
		TurnID:        m.activeTurnID,
	})
	m.activeApprovalTraceKey = key
	if m.activeTurnID != "" {
		m.executionTrace.setStatus("turn:"+m.activeTurnID+":working", TraceWaiting, false)
	}
}

func (m *Model) traceRuntimeFailure(ev runtime.Event) {
	key := ""
	if strings.TrimSpace(ev.ID) != "" {
		key = "runtime:" + strings.TrimSpace(ev.ID)
	}
	detail := ""
	if ev.Error != nil {
		detail = safeTraceText(ev.Error.Message)
	}
	if ev.RawSensitivity == runtime.SensitivitySensitive {
		detail = "[sensitive runtime detail hidden]"
	}
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceRuntime,
		Title:         "Runtime",
		Detail:        detail,
		Status:        TraceFailed,
		InvocationKey: key,
		SessionID:     runtime.SessionID(ev.SessionID),
		TurnID:        m.activeTurnID,
		FinishedAt:    time.Now(),
	})
}

func (m *Model) finishActiveTurnTrace(status TraceStatus) {
	if m.activeTurnID == "" {
		return
	}
	m.executionTrace.setStatus("turn:"+m.activeTurnID+":working", status, true)
	m.executionTrace.setStatus("turn:"+m.activeTurnID+":agent", status, true)
}

func (m *Model) traceStructuredMetadata(ev runtime.Event) {
	if len(ev.Metadata) == 0 {
		return
	}
	status, finished := structuredTraceStatus(ev.Type)
	for _, spec := range []struct {
		category TraceCategory
		nameKey  string
		idKey    string
	}{
		{TraceSkill, "skill", "skillInvocationID"},
		{TracePlugin, "plugin", "pluginInvocationID"},
		{TraceSubagent, "subagent", "subagentInvocationID"},
	} {
		name := safeTraceText(ev.Metadata[spec.nameKey])
		if name == "" {
			continue
		}
		id := strings.TrimSpace(ev.Metadata[spec.idKey])
		key := ""
		if id != "" {
			key = string(spec.category) + ":" + id
		}
		key = m.executionTrace.upsert(ExecutionTraceEntry{
			Category:      spec.category,
			Title:         name,
			Status:        status,
			InvocationKey: key,
			SessionID:     runtime.SessionID(ev.SessionID),
			TurnID:        m.activeTurnID,
		})
		if finished {
			m.executionTrace.setStatus(key, status, true)
		}
	}
}

func structuredTraceStatus(eventType runtime.EventType) (TraceStatus, bool) {
	switch eventType {
	case eventTypeToolSuccess, eventTypeStepFinished:
		return TraceSuccess, true
	case eventTypeToolFailed, eventTypeSessionError, eventTypeRuntimeError:
		return TraceFailed, true
	case eventTypeApprovalRequested:
		return TraceWaiting, false
	default:
		return TraceRunning, false
	}
}

// safeTraceDetail deliberately ignores Event.Content and Raw. Those fields may
// contain model/tool payloads. Task 25 renders only structured Codea metadata
// summaries, then truncates/redacts them again at the presentation boundary.
func safeTraceDetail(ev runtime.Event) string {
	if len(ev.Metadata) == 0 {
		return ""
	}
	keys := []string{"target"}
	if ev.RawSensitivity != runtime.SensitivitySensitive {
		keys = append(keys, "inputSummary", "outputSummary", "errorSummary")
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if value := safeTraceText(ev.Metadata[key]); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " · ")
}

func safeTraceText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = redactCommonSecret(value)
	runes := []rune(value)
	if len(runes) > 240 {
		value = string(runes[:240]) + "…"
	}
	return value
}

func redactCommonSecret(value string) string {
	lower := strings.ToLower(value)
	markers := []string{"api_key=", "api-key=", "apikey=", "password=", "token=", "secret=", "authorization:", "bearer ", "ghp_", "sk-"}
	for _, marker := range markers {
		searchFrom := 0
		for searchFrom < len(lower) {
			rel := strings.Index(lower[searchFrom:], marker)
			if rel < 0 {
				break
			}
			idx := searchFrom + rel
			start := idx + len(marker)
			end := start
			for end < len(value) && value[end] != ' ' && value[end] != '\t' && value[end] != '\n' && value[end] != '\r' && value[end] != ',' && value[end] != ';' {
				end++
			}
			value = value[:start] + "***" + value[end:]
			lower = strings.ToLower(value)
			searchFrom = start + len("***")
			if end == start {
				break
			}
		}
	}
	return value
}
