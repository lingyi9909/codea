package opencode

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventMapperGoldenSSE(t *testing.T) {
	goldenPath := filepath.Join("..", "..", "..", "runtime", "openapi", "golden-sse-s2.jsonl")
	f, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("failed to open golden SSE file: %v", err)
	}
	defer f.Close()

	var seq int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		seq++
		event, err := MapEvent([]byte(line), seq)
		if err != nil {
			t.Fatalf("map event %d: %v", seq, err)
		}
		if len(event.Raw) == 0 {
			t.Fatalf("event %d lost raw payload", seq)
		}
		if event.Type == "" {
			t.Fatalf("event %d has no mapped or unknown type", seq)
		}
		if event.RawType == "" {
			t.Fatalf("event %d has empty RawType", seq)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	if seq == 0 {
		t.Fatal("no events mapped from golden SSE")
	}
}

func TestEventMapperGoldenSSEEventTypes(t *testing.T) {
	goldenPath := filepath.Join("..", "..", "..", "runtime", "openapi", "golden-sse-s2.jsonl")
	f, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("failed to open golden SSE file: %v", err)
	}
	defer f.Close()

	seen := map[string]int{}
	scanner := bufio.NewScanner(f)
	var seq int64
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		seq++
		event, err := MapEvent([]byte(line), seq)
		if err != nil {
			t.Fatalf("map event %d: %v", seq, err)
		}
		seen[string(event.Type)]++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	if len(seen) == 0 {
		t.Fatal("no event types found")
	}
}

func TestEventMapperUnknownType(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"custom.unknown.event","properties":{"key":"value"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "custom.unknown.event" {
		t.Fatalf("expected Type=custom.unknown.event, got %q", event.Type)
	}
	if event.RawType != "custom.unknown.event" {
		t.Fatalf("expected RawType=custom.unknown.event, got %q", event.RawType)
	}
	if len(event.Raw) == 0 {
		t.Fatal("raw payload must not be empty for unknown event")
	}
}

func TestEventMapperMalformedJSON(t *testing.T) {
	raw := []byte(`not json`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "_unparseable_" {
		t.Fatalf("expected Type=_unparseable_, got %q", event.Type)
	}
	if string(event.Raw) != `not json` {
		t.Fatalf("expected Raw to preserve exact bytes, got %q", event.Raw)
	}
}

func TestEventMapperOversizedRaw(t *testing.T) {
	payload := strings.Repeat("x", 20*1024)
	raw := []byte(`{"directory":"/tmp","payload":{"type":"test","properties":{"data":"` + payload + `"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !event.RawTruncated {
		t.Fatal("expected RawTruncated=true for oversized payload")
	}
	if event.RawOriginalSize != len(raw) {
		t.Fatalf("expected RawOriginalSize=%d, got %d", len(raw), event.RawOriginalSize)
	}
	if len(event.Raw) > 16*1024 {
		t.Fatalf("expected Raw <= 16KB, got %d", len(event.Raw))
	}
}

func TestEventMapperExtractsSessionID(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses_abc123","status":{"type":"busy"}}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.SessionID != "ses_abc123" {
		t.Fatalf("expected SessionID=ses_abc123, got %q", event.SessionID)
	}
}

func TestEventMapperExtractsMessagePartIDs(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"hello"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.SessionID != "s1" {
		t.Fatalf("expected SessionID=s1, got %q", event.SessionID)
	}
	if event.MessageID != "m1" {
		t.Fatalf("expected MessageID=m1, got %q", event.MessageID)
	}
	if event.PartID != "p1" {
		t.Fatalf("expected PartID=p1, got %q", event.PartID)
	}
}

func TestEventMapperExtractsContentFromDelta(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"hello world"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Content != "hello world" {
		t.Fatalf("expected Content='hello world', got %q", event.Content)
	}
}

func TestEventMapperPreservesRawJSON(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"s1"}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rawJSON map[string]any
	if err := json.Unmarshal(event.Raw, &rawJSON); err != nil {
		t.Fatalf("Raw is not valid JSON: %v", err)
	}
}
