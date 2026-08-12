package runtime

import (
	"encoding/json"
	"fmt"
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

// RuntimeErrorKind classifies a RuntimeError for application-level handling.
type RuntimeErrorKind string

const (
	RuntimeErrorTransport    RuntimeErrorKind = "transport"
	RuntimeErrorAuth         RuntimeErrorKind = "auth"
	RuntimeErrorProtocol     RuntimeErrorKind = "protocol"
	RuntimeErrorIncompatible RuntimeErrorKind = "incompatible"
	RuntimeErrorRecovery     RuntimeErrorKind = "recovery"
	RuntimeErrorBackpressure RuntimeErrorKind = "backpressure"
	RuntimeErrorCancelled    RuntimeErrorKind = "cancelled"
)

// RuntimeError carries Runtime-level error information with classification.
type RuntimeError struct {
	Kind          RuntimeErrorKind `json:"kind"`
	Operation     string           `json:"operation"`
	Message       string           `json:"message"`
	Code          string           `json:"code,omitempty"`
	Retryable     bool             `json:"retryable"`
	Cause         error            `json:"-"`
	VendorDetails json.RawMessage  `json:"vendorDetails,omitempty"`
}

// Error implements the error interface.
func (e *RuntimeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s: %s", e.Kind, e.Operation, e.Message, e.Cause.Error())
	}
	return fmt.Sprintf("[%s] %s: %s", e.Kind, e.Operation, e.Message)
}

// Unwrap returns the underlying cause.
func (e *RuntimeError) Unwrap() error {
	return e.Cause
}

// Sensitivity controls how Raw event data is handled.
type Sensitivity string

const (
	SensitivityPublic    Sensitivity = "public"
	SensitivityInternal  Sensitivity = "internal"
	SensitivitySensitive Sensitivity = "sensitive"
)
