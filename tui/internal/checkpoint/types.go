package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Kind string

const (
	KindBaseline Kind = "baseline"
	KindManual   Kind = "manual"
	KindFinal    Kind = "final"
	KindSafety   Kind = "safety"
)

type ErrorCode string

const (
	CodeCheckpointUnavailable ErrorCode = "CHECKPOINT_UNAVAILABLE"
	CodeStateCorrupt          ErrorCode = "CHECKPOINT_STATE_CORRUPT"
	CodeInvalidCheckpoint     ErrorCode = "CHECKPOINT_INVALID_ID"
	CodeRestoreInterrupted    ErrorCode = "CHECKPOINT_RESTORE_INTERRUPTED"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func IsCode(err error, code ErrorCode) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

type SkippedPath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Checkpoint struct {
	ID        string        `json:"id"`
	TaskID    string        `json:"taskId,omitempty"`
	TurnID    string        `json:"turnId,omitempty"`
	Commit    string        `json:"commit"`
	Label     string        `json:"label,omitempty"`
	Kind      Kind          `json:"kind"`
	CreatedAt time.Time     `json:"createdAt"`
	Skipped   []SkippedPath `json:"skipped,omitempty"`
}

type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

type Runner interface {
	Run(ctx context.Context, args []string, stdin []byte) (Result, error)
}
