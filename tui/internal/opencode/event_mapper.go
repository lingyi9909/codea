package opencode

import (
	"encoding/json"
	"time"

	"codea/tui/internal/runtime"
)

const maxRawSize = 16 * 1024

// Codea semantic event types mapped from OpenCode vendor events.
// Unknown vendor events map to "raw" with RawType preserved.
const (
	CodeaEventRuntimeConnected  runtime.EventType = "runtime.connected"
	CodeaEventSessionStatus     runtime.EventType = "session.status"
	CodeaEventSessionCreated    runtime.EventType = "session.created"
	CodeaEventSessionUpdated    runtime.EventType = "session.updated"
	CodeaEventSessionDeleted    runtime.EventType = "session.deleted"
	CodeaEventSessionDiff       runtime.EventType = "session.diff"
	CodeaEventSessionError      runtime.EventType = "session.error"
	CodeaEventSessionCompacted  runtime.EventType = "session.compacted"
	CodeaEventMessageUpdated    runtime.EventType = "message.updated"
	CodeaEventMessageRemoved    runtime.EventType = "message.removed"
	CodeaEventPartUpdated       runtime.EventType = "part.updated"
	CodeaEventPartRemoved       runtime.EventType = "part.removed"
	CodeaEventAnswerDelta       runtime.EventType = "answer.delta"
	CodeaEventReasoningDelta    runtime.EventType = "reasoning.delta"
	CodeaEventStepStarted       runtime.EventType = "step.started"
	CodeaEventStepFinished      runtime.EventType = "step.finished"
	CodeaEventStepFailed        runtime.EventType = "step.failed"
	CodeaEventToolCalled        runtime.EventType = "tool.called"
	CodeaEventToolSuccess       runtime.EventType = "tool.success"
	CodeaEventToolFailed        runtime.EventType = "tool.failed"
	CodeaEventApprovalRequested runtime.EventType = "approval.requested"
	CodeaEventApprovalResolved  runtime.EventType = "approval.resolved"
	CodeaEventRaw               runtime.EventType = "raw"
)

// vendorToCodea maps OpenCode vendor event types to Codea semantic types.
var vendorToCodea = map[string]runtime.EventType{
	"server.connected":         CodeaEventRuntimeConnected,
	"server.instance.disposed": CodeaEventRuntimeConnected,
	"session.created":          CodeaEventSessionCreated,
	"session.updated":          CodeaEventSessionUpdated,
	"session.deleted":          CodeaEventSessionDeleted,
	"session.status":           CodeaEventSessionStatus,
	"session.diff":             CodeaEventSessionDiff,
	"session.error":            CodeaEventSessionError,
	"session.compacted":        CodeaEventSessionCompacted,
	"message.updated":          CodeaEventMessageUpdated,
	"message.removed":          CodeaEventMessageRemoved,
	"message.part.removed":     CodeaEventPartRemoved,
	"permission.asked":         CodeaEventApprovalRequested,
	"permission.replied":       CodeaEventApprovalResolved,
}

// sseEnvelope is the top-level SSE event envelope from OpenCode.
type sseEnvelope struct {
	Directory string          `json:"directory"`
	Payload   json.RawMessage `json:"payload"`
}

// ssePayload is the payload within the envelope.
type ssePayload struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

// sseCommonProps captures fields found across many event properties.
type sseCommonProps struct {
	SessionID string          `json:"sessionID"`
	MessageID string          `json:"messageID"`
	PartID    string          `json:"partID"`
	Field     string          `json:"field"`
	Delta     string          `json:"delta"`
	Time      float64         `json:"time"`
	Status    *sseStatus      `json:"status"`
	Info      json.RawMessage `json:"info"`
	Part      *ssePart        `json:"part"`
	Error     json.RawMessage `json:"error"`
}

type sseStatus struct {
	Type string `json:"type"`
}

type ssePart struct {
	ID        string   `json:"id"`
	MessageID string   `json:"messageID"`
	SessionID string   `json:"sessionID"`
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	Reason    string   `json:"reason"`
	Tool      string   `json:"tool"`
	CallID    string   `json:"callID"`
	Time      *sseTime `json:"time"`
}

type sseTime struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// sseSessionInfo extracts projectID from session-level events.
type sseSessionInfo struct {
	ProjectID string `json:"projectID"`
	ID        string `json:"id"`
}

// ssePermissionProps extracts permission request data from permission.asked events.
type ssePermissionProps struct {
	ID         string `json:"id"`
	Permission string `json:"permission"`
	Action     string `json:"action"`
	SessionID  string `json:"sessionID"`
}

// MapEvent maps a raw OpenCode SSE event to a Codea runtime.Event.
// Sequence is the monotonically increasing connection-level sequence number.
func MapEvent(raw []byte, sequence int64) (runtime.Event, error) {
	rawSize := len(raw)

	var envelope sseEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return unparseableEvent(raw, rawSize, sequence), nil
	}

	var payload ssePayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return unparseableEvent(raw, rawSize, sequence), nil
	}

	var props sseCommonProps
	_ = json.Unmarshal(payload.Properties, &props)

	codeaType := mapVendorType(payload.Type, &props)
	event := runtime.Event{
		Type:     codeaType,
		Sequence: sequence,
		RawType:  payload.Type,
	}

	// Raw payload with truncation
	rawPayload := trimRaw(raw)
	event.Raw = rawPayload
	if len(rawPayload) < rawSize {
		event.RawTruncated = true
		event.RawOriginalSize = rawSize
	}

	// Common fields
	if props.SessionID != "" {
		event.SessionID = props.SessionID
	}
	if props.MessageID != "" {
		event.MessageID = props.MessageID
	}
	if props.PartID != "" {
		event.PartID = props.PartID
	}
	if props.Time > 0 {
		event.CreatedAt = time.UnixMilli(int64(props.Time))
	}

	// Content from delta or part text
	if props.Delta != "" {
		event.Content = props.Delta
	} else if props.Part != nil && props.Part.Text != "" {
		event.Content = props.Part.Text
	}

	// ProjectID from session-level info
	if props.Info != nil {
		var info sseSessionInfo
		if err := json.Unmarshal(props.Info, &info); err == nil {
			if info.ProjectID != "" {
				event.ProjectID = info.ProjectID
			}
		}
	}

	// Domain data extraction
	extractApproval(&event, &payload, &props)
	extractTool(&event, &props)
	extractError(&event, &props)

	return event, nil
}

func extractApproval(event *runtime.Event, payload *ssePayload, props *sseCommonProps) {
	if event.Type != CodeaEventApprovalRequested {
		return
	}
	var perm ssePermissionProps
	_ = json.Unmarshal(payload.Properties, &perm)
	approval := &runtime.ApprovalRequest{
		ID: perm.ID,
	}
	if perm.Permission != "" {
		approval.Permission = perm.Permission
	} else if perm.Action != "" {
		approval.Permission = perm.Action
	}
	if approval.ID != "" || approval.Permission != "" {
		event.Approval = approval
	}
	if perm.SessionID != "" && event.SessionID == "" {
		event.SessionID = perm.SessionID
	}
}

func extractTool(event *runtime.Event, props *sseCommonProps) {
	if event.Type != CodeaEventToolCalled || props.Part == nil {
		return
	}
	name := props.Part.Tool
	callID := props.Part.CallID
	if callID == "" {
		callID = props.Part.ID
	}
	if name != "" || callID != "" {
		event.Tool = &runtime.ToolEvent{
			Name:   name,
			CallID: callID,
		}
	}
}

// sseErrorData extracts the message from a structured error object.
type sseErrorData struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

type sseErrorInner struct {
	Message string `json:"message"`
}

func extractError(event *runtime.Event, props *sseCommonProps) {
	if event.Type != CodeaEventSessionError && event.RawType != "_unparseable_" {
		return
	}
	if len(props.Error) == 0 {
		return
	}
	// Try string first
	var msg string
	if err := json.Unmarshal(props.Error, &msg); err == nil {
		event.Error = &runtime.RuntimeError{
			Code:    string(CodeaEventSessionError),
			Message: msg,
		}
		return
	}
	// Try structured error: {"name":"...","data":{"message":"..."}}
	var ed sseErrorData
	if err := json.Unmarshal(props.Error, &ed); err == nil {
		if ed.Data != nil {
			var inner sseErrorInner
			if err := json.Unmarshal(ed.Data, &inner); err == nil && inner.Message != "" {
				event.Error = &runtime.RuntimeError{
					Code:    ed.Name,
					Message: inner.Message,
				}
				return
			}
		}
		event.Error = &runtime.RuntimeError{
			Code:    ed.Name,
			Message: string(props.Error),
		}
	}
}

func mapVendorType(vendorType string, props *sseCommonProps) runtime.EventType {
	// message.part.delta → answer.delta or reasoning.delta based on field
	if vendorType == "message.part.delta" {
		if props.Field == "reasoning" {
			return CodeaEventReasoningDelta
		}
		return CodeaEventAnswerDelta
	}

	// message.part.updated → classify by part type
	if vendorType == "message.part.updated" && props.Part != nil {
		switch props.Part.Type {
		case "step-start":
			return CodeaEventStepStarted
		case "step-finish":
			return CodeaEventStepFinished
		case "tool":
			return CodeaEventToolCalled
		}
		return CodeaEventPartUpdated
	}

	// permission events map to approval domain
	if vendorType == "permission.v2.asked" {
		return CodeaEventApprovalRequested
	}
	if vendorType == "permission.v2.replied" {
		return CodeaEventApprovalResolved
	}

	if ct, ok := vendorToCodea[vendorType]; ok {
		return ct
	}
	return CodeaEventRaw
}

func unparseableEvent(raw []byte, rawSize int, sequence int64) runtime.Event {
	rawPayload := trimRaw(raw)
	return runtime.Event{
		Type:            "_unparseable_",
		Sequence:        sequence,
		Raw:             rawPayload,
		RawType:         "_unparseable_",
		RawTruncated:    len(rawPayload) < rawSize,
		RawOriginalSize: rawSize,
	}
}

func trimRaw(raw []byte) []byte {
	if len(raw) <= maxRawSize {
		return raw
	}
	return raw[:maxRawSize]
}
