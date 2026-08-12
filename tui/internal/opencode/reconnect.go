package opencode

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Backoff returns the duration to wait before the next reconnect attempt.
// The sequence is: 500ms, 1s, 2s, 5s (capped).
func Backoff(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 500 * time.Millisecond
	case 2:
		return 1 * time.Second
	case 3:
		return 2 * time.Second
	default:
		return 5 * time.Second
	}
}

// IsRetryableHTTP reports whether an HTTP error from Subscribe is retryable.
// 401/403 are not retryable without credential changes.
// 4xx client errors are protocol errors and not retryable.
// 5xx and transport errors (err != nil with no HTTP status) are retryable.
// Non-HTTP transport errors are retryable.
func IsRetryableHTTP(statusCode int, err error) bool {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return false
	}
	// Client errors (4xx) are protocol errors — not retryable.
	if statusCode >= 400 && statusCode < 500 {
		return false
	}
	// Transport-level error (no HTTP status) — retryable.
	if err != nil && statusCode == 0 {
		return true
	}
	return statusCode >= 500
}

// sseSubscribeFunc is the signature of SSEClient.Subscribe, extracted for testability.
type sseSubscribeFunc func(ctx context.Context) (<-chan SSERawEvent, error)

// ReconnectHook is called after a successful SSE reconnection and before
// draining live events. It returns events to emit before the live stream,
// typically used for recovery/compensation.
type ReconnectHook func(ctx context.Context) ([]SSERawEvent, error)

// ReconnectingSSEClient wraps an SSE subscribe function with automatic
// reconnection and backoff. It presents a single merged event stream to
// callers, transparently reconnecting on retryable failures.
type ReconnectingSSEClient struct {
	subscribe     sseSubscribeFunc
	reconnectHook ReconnectHook
}

// NewReconnectingSSEClient creates a reconnecting client backed by the given
// SSEClient.
func NewReconnectingSSEClient(c *SSEClient) *ReconnectingSSEClient {
	return &ReconnectingSSEClient{subscribe: c.Subscribe}
}

// SetReconnectHook sets an optional hook called after successful reconnect.
func (r *ReconnectingSSEClient) SetReconnectHook(hook ReconnectHook) {
	r.reconnectHook = hook
}

// newReconnectingClient is the test constructor that accepts an arbitrary subscribe function.
func newReconnectingClient(fn sseSubscribeFunc) *ReconnectingSSEClient {
	return &ReconnectingSSEClient{subscribe: fn}
}

// Subscribe opens an SSE stream and automatically reconnects on retryable
// failures with exponential backoff. The returned channel merges events from
// all reconnection attempts. It closes when ctx is cancelled or a non-retryable
// error occurs.
func (r *ReconnectingSSEClient) Subscribe(ctx context.Context) (<-chan SSERawEvent, error) {
	ch := make(chan SSERawEvent, 16)

	go func() {
		defer close(ch)

		attempt := 1
		var seq int64

		for {
			rawCh, err := r.subscribe(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				status := extractHTTPStatus(err)
				if !IsRetryableHTTP(status, err) {
					// Emit Auth RuntimeError for 401/403 before closing.
					if status == http.StatusUnauthorized || status == http.StatusForbidden {
						seq++
						ev := SSERawEvent{
							Data:     newRuntimeErrorEvent(err.Error(), "AUTH_ERROR"),
							Sequence: seq,
						}
						sendEvent(ctx, ch, ev)
					}
					return
				}
				// Emit a disconnect event before backoff.
				seq++
				ev := disconnectEvent(seq, err.Error(), "CONNECT_FAILED")
				if !sendEvent(ctx, ch, ev) {
					return
				}
				if !sleepBackoff(ctx, attempt) {
					return
				}
				attempt++
				continue
			}

			// Call reconnect hook if set (recovery/compensation).
			if r.reconnectHook != nil {
				hookEvents, hookErr := r.reconnectHook(ctx)
				if hookErr != nil {
					seq++
					ev := disconnectEvent(seq, hookErr.Error(), "RECOVERY_FAILED")
					if !sendEvent(ctx, ch, ev) {
						return
					}
				}
				for _, ev := range hookEvents {
					seq++
					ev.Sequence = seq
					if !sendEvent(ctx, ch, ev) {
						return
					}
				}
			}

			// Successfully connected — drain events and reset attempt counter.
			// Only reset after a full successful connection that produced at
			// least one event, preventing rapid connect/disconnect reset loops.
			connected := false
			for raw := range rawCh {
				connected = true
				seq++
				raw.Sequence = seq
				if !sendEvent(ctx, ch, raw) {
					return
				}
			}

			if connected {
				attempt = 1
			}

			// Stream ended. Check if context is done.
			if ctx.Err() != nil {
				return
			}

			// Stream ended without context cancellation — this is an abnormal
			// disconnect. Emit a disconnect event and attempt reconnection.
			seq++
			ev := disconnectEvent(seq, "SSE stream ended unexpectedly", "DISCONNECTED")
			if !sendEvent(ctx, ch, ev) {
				return
			}

			if !sleepBackoff(ctx, attempt) {
				return
			}
			attempt++
		}
	}()

	return ch, nil
}

func disconnectEvent(seq int64, msg, code string) SSERawEvent {
	return SSERawEvent{
		Data:     newRuntimeErrorEvent(msg, code),
		Sequence: seq,
	}
}

func sendEvent(ctx context.Context, ch chan<- SSERawEvent, ev SSERawEvent) bool {
	select {
	case ch <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

func sleepBackoff(ctx context.Context, attempt int) bool {
	d := Backoff(attempt)
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func extractHTTPStatus(err error) int {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode
	}
	// Fallback string matching for non-HTTPError errors.
	s := err.Error()
	if strings.Contains(s, "401") {
		return http.StatusUnauthorized
	}
	if strings.Contains(s, "403") {
		return http.StatusForbidden
	}
	return 0
}

// IsSSEDisconnect reports whether an SSERawEvent is a disconnect signal.
func IsSSEDisconnect(ev SSERawEvent) bool {
	return strings.Contains(string(ev.Data), `"DISCONNECTED"`) ||
		strings.Contains(string(ev.Data), `"CONNECT_FAILED"`) ||
		strings.Contains(string(ev.Data), `"SCANNER_ERROR"`)
}

// DisconnectReason extracts a human-readable reason from a disconnect event.
func DisconnectReason(ev SSERawEvent) string {
	return fmt.Sprintf("seq=%d: %s", ev.Sequence, string(ev.Data))
}
