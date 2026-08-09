package opencode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEClientSingleDataLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected Accept: text/event-stream, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		fmt.Fprintf(w, "data: {\"type\":\"test\"}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		if string(evt.Data) != `{"type":"test"}` {
			t.Fatalf("unexpected data: %s", evt.Data)
		}
		if evt.Sequence != 1 {
			t.Fatalf("expected sequence 1, got %d", evt.Sequence)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSSEClientMultipleDataLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: line1\n")
		fmt.Fprintf(w, "data: line2\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt := <-ch:
		if string(evt.Data) != "line1\nline2" {
			t.Fatalf("unexpected joined data: %s", evt.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSSEClientIgnoresComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, ": this is a comment\n")
		fmt.Fprintf(w, "data: real data\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt := <-ch:
		if string(evt.Data) != "real data" {
			t.Fatalf("unexpected data: %s", evt.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSSEClientMultipleEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: event1\n\n")
		fmt.Fprintf(w, "data: event2\n\n")
		fmt.Fprintf(w, "data: event3\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, want := range []string{"event1", "event2", "event3"} {
		select {
		case evt := <-ch:
			if string(evt.Data) != want {
				t.Fatalf("event %d: expected %q, got %q", i, want, evt.Data)
			}
			if evt.Sequence != int64(i+1) {
				t.Fatalf("event %d: expected sequence %d, got %d", i, i+1, evt.Sequence)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for event %d", i)
		}
	}
}

func TestSSEClientNon200Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Subscribe(ctx)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestSSEClientContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: event1\n\n")
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read first event
	select {
	case <-ch:
		// ok
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first event")
	}

	// Wait for context cancellation to close the channel
	select {
	case _, ok := <-ch:
		if ok {
			// might get one more event before close
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after context cancellation")
	}
}

func TestSSEClientLargePayload(t *testing.T) {
	largeData := strings.Repeat("x", 128*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: %s\n\n", largeData)
		flusher.Flush()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt := <-ch:
		if string(evt.Data) != largeData {
			t.Fatalf("large payload mismatch: got %d bytes, want %d bytes", len(evt.Data), len(largeData))
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for large event")
	}
}

func TestSSEClientBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "testuser" || pass != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: ok\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := NewSSEClient(srv.URL, "testuser", "testpass")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := client.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt := <-ch:
		if string(evt.Data) != "ok" {
			t.Fatalf("unexpected data: %s", evt.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}
