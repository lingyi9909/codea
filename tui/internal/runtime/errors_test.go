package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestRuntimeErrorKindConstants(t *testing.T) {
	kinds := []RuntimeErrorKind{
		RuntimeErrorTransport,
		RuntimeErrorAuth,
		RuntimeErrorProtocol,
		RuntimeErrorIncompatible,
		RuntimeErrorRecovery,
		RuntimeErrorBackpressure,
		RuntimeErrorCancelled,
	}
	for _, k := range kinds {
		if k == "" {
			t.Error("empty RuntimeErrorKind")
		}
	}
}

func TestNewTransportError(t *testing.T) {
	cause := fmt.Errorf("dial tcp 127.0.0.1:4141: connection refused")
	err := NewTransportError("Subscribe", "connection refused", cause)

	if err.Kind != RuntimeErrorTransport {
		t.Errorf("expected Kind=transport, got %s", err.Kind)
	}
	if err.Operation != "Subscribe" {
		t.Errorf("expected Operation=Subscribe, got %s", err.Operation)
	}
	if err.Message != "connection refused" {
		t.Errorf("unexpected Message: %s", err.Message)
	}
	if !err.Retryable {
		t.Error("transport errors should be retryable")
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
	if err.Error() == "" {
		t.Error("Error() should not be empty")
	}
}

func TestNewAuthError(t *testing.T) {
	cause := fmt.Errorf("HTTP 401")
	err := NewAuthError("Health", "unauthorized", cause)

	if err.Kind != RuntimeErrorAuth {
		t.Errorf("expected Kind=auth, got %s", err.Kind)
	}
	if err.Retryable {
		t.Error("auth errors should not be retryable without credential change")
	}
}

func TestNewAuthError403(t *testing.T) {
	cause := fmt.Errorf("HTTP 403 Forbidden")
	err := NewAuthError("CreateSession", "forbidden", cause)

	if err.Kind != RuntimeErrorAuth {
		t.Errorf("expected Kind=auth, got %s", err.Kind)
	}
	if err.Retryable {
		t.Error("403 should not be retryable")
	}
}

func TestNewProtocolError(t *testing.T) {
	cause := fmt.Errorf("invalid character")
	err := NewProtocolError("SSE", "malformed SSE data", cause)

	if err.Kind != RuntimeErrorProtocol {
		t.Errorf("expected Kind=protocol, got %s", err.Kind)
	}
	if err.Retryable {
		t.Error("protocol errors should not be retryable")
	}
}

func TestNewIncompatibleError(t *testing.T) {
	err := NewIncompatibleError("Capabilities", "required capability missing: sessions")

	if err.Kind != RuntimeErrorIncompatible {
		t.Errorf("expected Kind=incompatible, got %s", err.Kind)
	}
	if err.Retryable {
		t.Error("incompatible errors should not be retryable")
	}
}

func TestNewRecoveryError(t *testing.T) {
	cause := fmt.Errorf("history API returned 500")
	err := NewRecoveryError("HistoryCompensation", "failed to fetch message history", cause)

	if err.Kind != RuntimeErrorRecovery {
		t.Errorf("expected Kind=recovery, got %s", err.Kind)
	}
	if !err.Retryable {
		t.Error("recovery errors should be retryable")
	}
}

func TestNewBackpressureError(t *testing.T) {
	err := NewBackpressureError("EventChannel", "channel full: 256/256", nil)

	if err.Kind != RuntimeErrorBackpressure {
		t.Errorf("expected Kind=backpressure, got %s", err.Kind)
	}
	if err.Retryable {
		t.Error("backpressure errors should not be retryable")
	}
}

func TestNewCancelledError(t *testing.T) {
	cause := contextCanceledError()
	err := NewCancelledError("Subscribe", "context cancelled", cause)

	if err.Kind != RuntimeErrorCancelled {
		t.Errorf("expected Kind=cancelled, got %s", err.Kind)
	}
	if err.Retryable {
		t.Error("cancelled errors should not be retryable")
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cancel cause")
	}
}

func TestRuntimeErrorVendorDetails(t *testing.T) {
	raw := json.RawMessage(`{"status":401,"body":"Unauthorized"}`)
	err := NewAuthError("Health", "unauthorized", fmt.Errorf("HTTP 401"))
	err.VendorDetails = raw

	if string(err.VendorDetails) != `{"status":401,"body":"Unauthorized"}` {
		t.Errorf("VendorDetails mismatch: %s", err.VendorDetails)
	}
}

func TestRuntimeErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := NewTransportError("Subscribe", "connection failed", cause)

	var target *RuntimeError
	if !errors.As(err, &target) {
		t.Error("errors.As should match *RuntimeError")
	}
	if target.Kind != RuntimeErrorTransport {
		t.Errorf("errors.As returned wrong Kind: %s", target.Kind)
	}
}

func TestRuntimeErrorErrorMethod(t *testing.T) {
	tests := []struct {
		name string
		err  *RuntimeError
		want string
	}{
		{
			name: "with cause",
			err:  NewTransportError("Subscribe", "connection refused", fmt.Errorf("dial tcp: connection refused")),
			want: "[transport] Subscribe: connection refused: dial tcp: connection refused",
		},
		{
			name: "without cause",
			err:  NewIncompatibleError("Capabilities", "required capability missing"),
			want: "[incompatible] Capabilities: required capability missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.want {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}

func TestErrorIsChecks(t *testing.T) {
	transportErr := NewTransportError("Subscribe", "refused", nil)
	authErr := NewAuthError("Health", "unauthorized", nil)

	if !IsTransport(transportErr) {
		t.Error("IsTransport should match transport error")
	}
	if IsTransport(authErr) {
		t.Error("IsTransport should not match auth error")
	}
	if !IsAuth(authErr) {
		t.Error("IsAuth should match auth error")
	}
	if !IsRetryable(transportErr) {
		t.Error("transport error should be retryable")
	}
	if IsRetryable(authErr) {
		t.Error("auth error should not be retryable")
	}
}

func TestErrorClassificationFromError(t *testing.T) {
	// Application should classify errors by kind, not by string matching.
	wrapped := fmt.Errorf("wrapper: %w", NewTransportError("Health", "refused", nil))

	if !IsTransport(wrapped) {
		t.Error("IsTransport should work through fmt.Errorf wrapping")
	}

	var rErr *RuntimeError
	if !errors.As(wrapped, &rErr) {
		t.Error("errors.As should extract RuntimeError through wrapping")
	}
	if rErr.Operation != "Health" {
		t.Errorf("expected Operation=Health, got %s", rErr.Operation)
	}
}

// contextCanceledError returns an error that satisfies context.Canceled-like
// semantics, avoiding importing "context" just for the sentinel.
func contextCanceledError() error {
	return &canceledError{}
}

type canceledError struct{}

func (e *canceledError) Error() string { return "context canceled" }
