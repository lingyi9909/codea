package runtime

import "errors"

// NewTransportError creates a transport-level RuntimeError (e.g., connection
// refused, DNS resolution failure, network timeout). Transport errors are
// retryable.
func NewTransportError(op, msg string, cause error) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorTransport,
		Operation: op,
		Message:   msg,
		Cause:     cause,
		Retryable: true,
	}
}

// NewAuthError creates an authentication/authorization RuntimeError (e.g.,
// HTTP 401, 403). Auth errors are not retryable without credential changes.
func NewAuthError(op, msg string, cause error) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorAuth,
		Operation: op,
		Message:   msg,
		Cause:     cause,
		Retryable: false,
	}
}

// NewProtocolError creates a protocol-level RuntimeError (e.g., malformed
// SSE, invalid JSON, unexpected response structure). Not retryable.
func NewProtocolError(op, msg string, cause error) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorProtocol,
		Operation: op,
		Message:   msg,
		Cause:     cause,
		Retryable: false,
	}
}

// NewIncompatibleError creates a compatibility RuntimeError (e.g., required
// capability missing, version mismatch). Not retryable.
func NewIncompatibleError(op, msg string) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorIncompatible,
		Operation: op,
		Message:   msg,
		Retryable: false,
	}
}

// NewRecoveryError creates a recovery RuntimeError (e.g., history API
// failure during state compensation). Retryable — recovery can be attempted
// again.
func NewRecoveryError(op, msg string, cause error) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorRecovery,
		Operation: op,
		Message:   msg,
		Cause:     cause,
		Retryable: true,
	}
}

// NewBackpressureError creates a backpressure RuntimeError (e.g., event
// channel full). Not retryable — the current stream is unsafe and should be
// torn down.
func NewBackpressureError(op, msg string, cause error) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorBackpressure,
		Operation: op,
		Message:   msg,
		Cause:     cause,
		Retryable: false,
	}
}

// NewCancelledError creates a cancellation RuntimeError. Not retryable.
func NewCancelledError(op, msg string, cause error) *RuntimeError {
	return &RuntimeError{
		Kind:      RuntimeErrorCancelled,
		Operation: op,
		Message:   msg,
		Cause:     cause,
		Retryable: false,
	}
}

// IsTransport reports whether err is or wraps a RuntimeError with Kind Transport.
func IsTransport(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorTransport
}

// IsAuth reports whether err is or wraps a RuntimeError with Kind Auth.
func IsAuth(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorAuth
}

// IsProtocol reports whether err is or wraps a RuntimeError with Kind Protocol.
func IsProtocol(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorProtocol
}

// IsIncompatible reports whether err is or wraps a RuntimeError with Kind Incompatible.
func IsIncompatible(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorIncompatible
}

// IsRecovery reports whether err is or wraps a RuntimeError with Kind Recovery.
func IsRecovery(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorRecovery
}

// IsBackpressure reports whether err is or wraps a RuntimeError with Kind Backpressure.
func IsBackpressure(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorBackpressure
}

// IsCancelled reports whether err is or wraps a RuntimeError with Kind Cancelled.
func IsCancelled(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Kind == RuntimeErrorCancelled
}

// IsRetryable reports whether err is a RuntimeError that can be retried.
func IsRetryable(err error) bool {
	var r *RuntimeError
	return errors.As(err, &r) && r.Retryable
}
