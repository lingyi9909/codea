package opencode

import (
	"encoding/json"
	"strings"
	"time"

	"codea/tui/internal/runtime"
)

const maxRawSize = 16 * 1024

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
	CodeaEventRuntimeError      runtime.EventType = "runtime.error"
	CodeaEventRaw               runtime.EventType = "raw"
)

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
	"runtime_error":            CodeaEventRuntimeError,
}

type sseEnvelope struct {
	Directory string          `json:"directory"`
	Payload   json.RawMessage `json:"payload"`
}

type ssePayload struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

type sseCommonProps struct {
	SessionID    string          `json:"sessionID"`
	MessageID    string          `json:"messageID"`
	PartID       string          `json:"partID"`
	Field        string          `json:"field"`
	Delta        string          `json:"delta"`
	Time         float64         `json:"time"`
	Status       *sseStatus      `json:"status"`
	Info         json.RawMessage `json:"info"`
	Part         *ssePart        `json:"part"`
	CallID       string          `json:"callID"`
	Tool         string          `json:"tool"`
	Error        json.RawMessage `json:"error"`
	Code         string          `json:"code"`
	Partial      string          `json:"partial"`
	OriginalSize int             `json:"originalSize"`
}

type sseStatus struct {
	Type string `json:"type"`
}

type ssePart struct {
	ID          string        `json:"id"`
	MessageID   string        `json:"messageID"`
	SessionID   string        `json:"sessionID"`
	Type        string        `json:"type"`
	Text        string        `json:"text"`
	Reason      string        `json:"reason"`
	Tool        string        `json:"tool"`
	CallID      string        `json:"callID"`
	Agent       string        `json:"agent"`
	Description string        `json:"description"`
	State       *ssePartState `json:"state"`
	Time        *sseTime      `json:"time"`
}

type ssePartState struct {
	Status   string         `json:"status"`
	Input    map[string]any `json:"input"`
	Metadata map[string]any `json:"metadata"`
}

type sseTime struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type sseSessionInfo struct {
	ProjectID string `json:"projectID"`
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	ParentID  string `json:"parentID"`
}

type ssePermissionProps struct {
	ID         string         `json:"id"`
	Permission string         `json:"permission"`
	Action     string         `json:"action"`
	SessionID  string         `json:"sessionID"`
	Patterns   []string       `json:"patterns"`
	Metadata   map[string]any `json:"metadata"`
}

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
	event := runtime.Event{Type: codeaType, Sequence: sequence, RawType: payload.Type}

	rawPayload := trimRaw(raw)
	event.Raw = rawPayload
	if len(rawPayload) < rawSize {
		event.RawTruncated = true
		event.RawOriginalSize = rawSize
	}

	if props.SessionID != "" {
		event.SessionID = props.SessionID
	}
	if props.MessageID != "" {
		event.MessageID = props.MessageID
	}
	if props.PartID != "" {
		event.PartID = props.PartID
	}
	if props.Part != nil {
		if event.SessionID == "" && props.Part.SessionID != "" {
			event.SessionID = props.Part.SessionID
		}
		if event.MessageID == "" && props.Part.MessageID != "" {
			event.MessageID = props.Part.MessageID
		}
		if event.PartID == "" && props.Part.ID != "" {
			event.PartID = props.Part.ID
		}
	}
	if props.Time > 0 {
		event.CreatedAt = time.UnixMilli(int64(props.Time))
	}

	if props.Delta != "" {
		event.Content = props.Delta
	} else if props.Part != nil && props.Part.Text != "" {
		event.Content = props.Part.Text
	}

	if props.Info != nil {
		var info sseSessionInfo
		if err := json.Unmarshal(props.Info, &info); err == nil {
			if info.ProjectID != "" {
				event.ProjectID = info.ProjectID
			}
			if payload.Type == "message.updated" {
				if event.SessionID == "" && info.SessionID != "" {
					event.SessionID = info.SessionID
				}
				if event.MessageID == "" && info.ID != "" {
					event.MessageID = info.ID
				}
				event.MessageRole = strings.TrimSpace(info.Role)
				event.ParentMessageID = strings.TrimSpace(info.ParentID)
			}
		}
	}

	extractApproval(&event, &payload, &props)
	extractTool(&event, &props)
	extractExecutionEvidence(&event, &props)
	extractError(&event, &props)

	return event, nil
}

func extractApproval(event *runtime.Event, payload *ssePayload, props *sseCommonProps) {
	if event.Type != CodeaEventApprovalRequested {
		return
	}
	var perm ssePermissionProps
	_ = json.Unmarshal(payload.Properties, &perm)
	approval := &runtime.ApprovalRequest{ID: perm.ID}
	if perm.Permission != "" {
		approval.Permission = perm.Permission
	} else if perm.Action != "" {
		approval.Permission = perm.Action
	}
	approval.Command = permissionCommand(perm)
	if approval.ID != "" || approval.Permission != "" {
		event.Approval = approval
	}
	if perm.SessionID != "" && event.SessionID == "" {
		event.SessionID = perm.SessionID
	}
}

func permissionCommand(perm ssePermissionProps) string {
	if cmd, ok := perm.Metadata["command"].(string); ok && cmd != "" {
		return cmd
	}
	if len(perm.Patterns) > 0 {
		return strings.Join(perm.Patterns, " ")
	}
	return ""
}

var taskToolMetadataAllowlist = []string{
	"codeaTaskPlan",
	"codeaPlanTotal",
	"codeaPlanCompleted",
	"codeaPlanActive",
}

func safeTaskToolMetadata(props *sseCommonProps) map[string]string {
	if props.Part == nil || props.Part.State == nil || len(props.Part.State.Metadata) == 0 {
		return nil
	}
	metadata := make(map[string]string)
	for _, key := range taskToolMetadataAllowlist {
		if value := mapString(props.Part.State.Metadata, key); value != "" || key == "codeaPlanActive" {
			if _, exists := props.Part.State.Metadata[key]; exists {
				metadata[key] = value
			}
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func extractTool(event *runtime.Event, props *sseCommonProps) {
	switch event.Type {
	case CodeaEventToolCalled, CodeaEventToolSuccess, CodeaEventToolFailed:
		name := props.Tool
		callID := props.CallID
		if props.Part != nil {
			if name == "" {
				name = props.Part.Tool
			}
			if callID == "" {
				callID = props.Part.CallID
			}
			if callID == "" {
				callID = props.Part.ID
			}
		}
		if name != "" || callID != "" {
			event.Tool = &runtime.ToolEvent{
				Name:     name,
				CallID:   callID,
				Metadata: safeTaskToolMetadata(props),
			}
		}
	}
}

func extractExecutionEvidence(event *runtime.Event, props *sseCommonProps) {
	part := props.Part
	if part == nil {
		return
	}
	metadata := make(map[string]string)

	if part.Type == "tool" && part.Tool == "skill" && part.State != nil {
		if skill := mapString(part.State.Input, "name"); skill != "" {
			metadata["skill"] = skill
			if id := firstNonEmpty(part.CallID, part.ID); id != "" {
				metadata["skillInvocationID"] = id
			}
		}
	}

	if part.State != nil {
		if plugin := mapString(part.State.Metadata, "codeaPlugin"); plugin != "" {
			metadata["plugin"] = plugin
			id := mapString(part.State.Metadata, "codeaPluginInvocationID")
			if id == "" {
				id = firstNonEmpty(part.CallID, part.ID)
			}
			if id != "" {
				metadata["pluginInvocationID"] = id
			}
		}
	}

	if part.Type == "subtask" {
		if subagent := strings.TrimSpace(part.Agent); subagent != "" {
			metadata["subagent"] = subagent
			if id := strings.TrimSpace(part.ID); id != "" {
				metadata["subagentInvocationID"] = id
			}
		}
	}

	if len(metadata) > 0 {
		event.Metadata = metadata
	}
}

func mapString(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

type sseErrorData struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

type sseErrorInner struct {
	Message string `json:"message"`
}

func extractError(event *runtime.Event, props *sseCommonProps) {
	switch {
	case event.Type == CodeaEventRuntimeError:
		code := props.Code
		if code == "" {
			code = string(CodeaEventRuntimeError)
		}
		var msg string
		if len(props.Error) > 0 {
			_ = json.Unmarshal(props.Error, &msg)
		}
		event.Error = &runtime.RuntimeError{
			Kind:          runtimeErrorKindFromCode(code),
			Operation:     "EventMap",
			Code:          code,
			Message:       msg,
			VendorDetails: props.Error,
		}
		if props.Partial != "" {
			raw := []byte(props.Partial)
			event.Raw = trimRaw(raw)
			if props.OriginalSize > 0 && len(raw) < props.OriginalSize {
				event.RawTruncated = true
				event.RawOriginalSize = props.OriginalSize
			}
		}

	case event.Type == CodeaEventSessionError:
		if len(props.Error) == 0 {
			return
		}
		var msg string
		if err := json.Unmarshal(props.Error, &msg); err == nil {
			event.Error = &runtime.RuntimeError{
				Kind:          runtime.RuntimeErrorProtocol,
				Operation:     "EventMap",
				Code:          string(CodeaEventSessionError),
				Message:       msg,
				VendorDetails: props.Error,
			}
			return
		}
		var ed sseErrorData
		if err := json.Unmarshal(props.Error, &ed); err == nil {
			if ed.Data != nil {
				var inner sseErrorInner
				if err := json.Unmarshal(ed.Data, &inner); err == nil && inner.Message != "" {
					event.Error = &runtime.RuntimeError{
						Kind:          runtime.RuntimeErrorProtocol,
						Operation:     "EventMap",
						Code:          ed.Name,
						Message:       inner.Message,
						VendorDetails: props.Error,
					}
					return
				}
			}
			event.Error = &runtime.RuntimeError{
				Kind:          runtime.RuntimeErrorProtocol,
				Operation:     "EventMap",
				Code:          ed.Name,
				Message:       string(props.Error),
				VendorDetails: props.Error,
			}
		}

	case event.RawType == "_unparseable_":
		if len(props.Error) == 0 {
			return
		}
		var msg string
		if err := json.Unmarshal(props.Error, &msg); err == nil {
			event.Error = &runtime.RuntimeError{
				Kind:          runtime.RuntimeErrorProtocol,
				Operation:     "EventMap",
				Code:          string(CodeaEventSessionError),
				Message:       msg,
				VendorDetails: props.Error,
			}
		}
	}
}

func mapVendorType(vendorType string, props *sseCommonProps) runtime.EventType {
	if vendorType == "message.part.delta" {
		if props.Field == "reasoning" {
			return CodeaEventReasoningDelta
		}
		return CodeaEventAnswerDelta
	}

	if vendorType == "message.part.updated" && props.Part != nil {
		switch props.Part.Type {
		case "step-start":
			return CodeaEventStepStarted
		case "step-finish":
			return CodeaEventStepFinished
		case "tool":
			if props.Part.State != nil {
				switch props.Part.State.Status {
				case "completed":
					return CodeaEventToolSuccess
				case "error":
					return CodeaEventToolFailed
				case "pending", "running":
					return CodeaEventToolCalled
				}
			}
			return CodeaEventToolCalled
		}
		return CodeaEventPartUpdated
	}

	if vendorType == "permission.v2.asked" {
		return CodeaEventApprovalRequested
	}
	if vendorType == "permission.v2.replied" {
		return CodeaEventApprovalResolved
	}

	if vendorType == "session.next.tool.called" {
		return CodeaEventToolCalled
	}
	if vendorType == "session.next.tool.success" {
		return CodeaEventToolSuccess
	}
	if vendorType == "session.next.tool.failed" {
		return CodeaEventToolFailed
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

func runtimeErrorKindFromCode(code string) runtime.RuntimeErrorKind {
	switch code {
	case "AUTH_ERROR", "UNAUTHORIZED", "FORBIDDEN":
		return runtime.RuntimeErrorAuth
	case "DISCONNECTED", "CONNECT_FAILED", "SCANNER_ERROR", "TRANSPORT_ERROR":
		return runtime.RuntimeErrorTransport
	default:
		return runtime.RuntimeErrorProtocol
	}
}

func trimRaw(raw []byte) []byte {
	if len(raw) <= maxRawSize {
		return raw
	}
	return raw[:maxRawSize]
}
