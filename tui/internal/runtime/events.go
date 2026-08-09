package runtime

import (
	"encoding/json"
	"time"
)

// EventType classifies a Runtime event.
type EventType string

// Event represents a single Runtime event with mapped fields and Raw payload.
type Event struct {
	ID              string
	Type            EventType
	Sequence        int64
	ProjectID       string
	SessionID       string
	MessageID       string
	PartID          string
	CreatedAt       time.Time
	Content         string
	Tool            *ToolEvent
	Approval        *ApprovalRequest
	Error           *RuntimeError
	Metadata        map[string]string
	RawType         string
	Raw             json.RawMessage
	RawSensitivity  Sensitivity
	RawTruncated    bool
	RawOriginalSize int
}

// ToolEvent carries tool lifecycle data within an Event.
type ToolEvent struct {
	Name   string
	CallID string
}

// ApprovalRequest carries a permission request within an Event.
type ApprovalRequest struct {
	ID         string
	Permission string
}

// RuntimeError carries Runtime-level error information.
type RuntimeError struct {
	Code    string
	Message string
}

// Sensitivity controls how Raw event data is handled.
type Sensitivity string

const (
	SensitivityPublic   Sensitivity = "public"
	SensitivityInternal Sensitivity = "internal"
	SensitivityPrivate  Sensitivity = "private"
)
