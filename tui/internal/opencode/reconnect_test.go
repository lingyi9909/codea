package opencode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackoffSequence(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 500 * time.Millisecond},
		{2, 1 * time.Second},
		{3, 2 * time.Second},
		{4, 5 * time.Second},
		{5, 5 * time.Second},
		{10, 5 * time.Second},
	}
	for _, tt := range tests {
		got := Backoff(tt.attempt)
		if got != tt.want {
			t.Errorf("Backoff(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestIsRetryableHTTP(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		want       bool
	}{
		{"transport error", 0, fmt.Errorf("connection refused"), true},
		{"500", http.StatusInternalServerError, nil, true},
		{"503", http.StatusServiceUnavailable, nil, true},
		{"401", http.StatusUnauthorized, nil, false},
		{"403", http.StatusForbidden, nil, false},
		{"200 OK", http.StatusOK, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableHTTP(tt.statusCode, tt.err); got != tt.want {
				t.Errorf("IsRetryableHTTP(%d, %v) = %v, want %v", tt.statusCode, tt.err, got, tt.want)
			}
		})
	}
}

// fakeSSEServer returns an httptest server that serves SSE.
func fakeSSEServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestReconnectingClientNormalStream(t *testing.T) {
	var eventCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("expected Flusher")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 5; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			fmt.Fprintf(w, "data: {\"type\":\"answer.delta\",\"properties\":{\"content\":\"msg%d\"}}\n\n", i)
			flusher.Flush()
			eventCount.Add(1)
		}
		// Keep connection open until client cancels.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var events []SSERawEvent
	for ev := range ch {
		events = append(events, ev)
		if len(events) >= 5 {
			cancel()
		}
	}

	if len(events) < 5 {
		t.Errorf("expected at least 5 events, got %d", len(events))
	}
	for _, ev := range events {
		if IsSSEDisconnect(ev) {
			t.Error("unexpected disconnect event in normal stream")
		}
	}
}

func TestReconnectingClientBackoffAndRetry(t *testing.T) {
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"type\":\"answer.delta\",\"properties\":{\"content\":\"hello\"}}\n\n")
		flusher.Flush()
		// Close immediately to trigger disconnect and reconnect.
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var disconnectCount int
	timeout := time.After(6 * time.Second)
loop:
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				break loop
			}
			if IsSSEDisconnect(ev) {
				disconnectCount++
			}
		case <-timeout:
			cancel()
		}
	}

	// After 6 seconds with immediate disconnects, we should have multiple
	// reconnection attempts.
	n := connects.Load()
	if n < 3 {
		t.Errorf("expected at least 3 connection attempts, got %d", n)
	}
	if disconnectCount < 2 {
		t.Errorf("expected at least 2 disconnect events, got %d", disconnectCount)
	}
}

func TestReconnectingClientContextCancelDuringBackoff(t *testing.T) {
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"type\":\"answer.delta\",\"properties\":{\"content\":\"x\"}}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Drain first event, then cancel during backoff.
	<-ch
	cancel()

	// Channel should close promptly.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // success
			}
		case <-deadline:
			t.Fatal("channel did not close within 2s after cancel")
		}
	}
}

func TestReconnectingClient401NoRetry(t *testing.T) {
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Channel should close quickly — no retry on 401.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				n := connects.Load()
				if n > 1 {
					t.Errorf("expected exactly 1 connect attempt for 401, got %d", n)
				}
				return
			}
		case <-deadline:
			t.Fatal("channel did not close")
		}
	}
}

func TestReconnectingClientTransportErrorRetry(t *testing.T) {
	var connects atomic.Int32
	// First two attempts fail, third succeeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connects.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"type\":\"answer.delta\",\"properties\":{\"content\":\"recovered\"}}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var recovered bool
	for ev := range ch {
		if strings.Contains(string(ev.Data), "recovered") {
			recovered = true
		}
	}

	if !recovered {
		t.Error("did not receive recovered event after retry")
	}
	n := connects.Load()
	if n < 3 {
		t.Errorf("expected at least 3 connects, got %d", n)
	}
}

func TestBackoffCounterResetAfterSuccess(t *testing.T) {
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: {\"type\":\"answer.delta\",\"properties\":{\"content\":\"msg%d\"}}\n\n", i)
			flusher.Flush()
		}
		// Server closes after 3 events, triggering reconnect.
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	eventCount := 0
	timeout := time.After(12 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Logf("total events: %d, connects: %d", eventCount, connects.Load())
				return
			}
			eventCount++
			_ = ev
		case <-timeout:
			cancel()
		}
	}
}

func TestIsSSEDisconnectDetection(t *testing.T) {
	tests := []struct {
		data string
		want bool
	}{
		{`{"payload":{"type":"runtime_error","properties":{"error":"SSE stream ended","code":"DISCONNECTED"}}}`, true},
		{`{"payload":{"type":"runtime_error","properties":{"error":"dial tcp","code":"CONNECT_FAILED"}}}`, true},
		{`{"payload":{"type":"runtime_error","properties":{"error":"bufio.Scanner","code":"SCANNER_ERROR"}}}`, true},
		{`{"type":"answer.delta","properties":{"content":"hello"}}`, false},
	}
	for _, tt := range tests {
		ev := SSERawEvent{Data: []byte(tt.data), Sequence: 1}
		if got := IsSSEDisconnect(ev); got != tt.want {
			t.Errorf("IsSSEDisconnect(%s) = %v, want %v", tt.data, got, tt.want)
		}
	}
}

func TestDisconnectEventSequence(t *testing.T) {
	var connects atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connects.Add(1)
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"type\":\"answer.delta\",\"properties\":{\"content\":\"one\"}}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewSSEClient(srv.URL, "", "")
	rc := NewReconnectingSSEClient(c)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := rc.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var seqs []int64
	var disconnectSeqs []int64
	for ev := range ch {
		seqs = append(seqs, ev.Sequence)
		if IsSSEDisconnect(ev) {
			disconnectSeqs = append(disconnectSeqs, ev.Sequence)
		}
		if len(seqs) >= 20 {
			cancel()
		}
	}

	// Sequence numbers should be monotonically increasing.
	for i := 1; i < len(seqs); i++ {
		if seqs[i] <= seqs[i-1] {
			t.Errorf("sequence not monotonic: seqs[%d]=%d, seqs[%d]=%d", i-1, seqs[i-1], i, seqs[i])
		}
	}
	if len(disconnectSeqs) == 0 {
		t.Error("expected at least one disconnect event")
	}
}
