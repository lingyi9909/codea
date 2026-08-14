package parity_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

// TestGoldenEventsNoSilentDrop is the Task 9 zero-silent-drop gate. It maps every
// non-empty record in the Task 1 Golden SSE capture through the OpenCode event
// mapper and asserts that each one lands in a semantic Codea event OR a raw
// passthrough event — never a silent drop. The result is reported as a
// total/semantic/raw breakdown so the gate is auditable rather than opaque.
func TestGoldenEventsNoSilentDrop(t *testing.T) {
	goldenPath := filepath.Join("..", "..", "..", "runtime", "openapi", "golden-sse-s2.jsonl")
	f, err := os.Open(goldenPath)
	if err != nil {
		t.Fatalf("open golden SSE: %v", err)
	}
	defer f.Close()

	var total, semantic, raw, silentDrop int
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	var seq int64
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		seq++
		total++
		ev, err := opencode.MapEvent([]byte(line), seq)
		if err != nil {
			t.Fatalf("map event %d: %v", seq, err)
		}
		if ev.Type == "" {
			silentDrop++
			t.Errorf("event %d silent drop: empty Type", seq)
			continue
		}
		if len(ev.Raw) == 0 {
			silentDrop++
			t.Errorf("event %d silent drop: empty Raw", seq)
			continue
		}
		if ev.RawType == "" {
			silentDrop++
			t.Errorf("event %d silent drop: empty RawType", seq)
			continue
		}
		if ev.Type == runtime.EventType("raw") || ev.Type == runtime.EventType("_unparseable_") {
			raw++
		} else {
			semantic++
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan golden SSE: %v", err)
	}
	if total == 0 {
		t.Fatal("no events in golden SSE")
	}
	if silentDrop != 0 {
		t.Errorf("silent drop = %d, want 0", silentDrop)
	}
	t.Logf("Golden SSE total=%d semantic=%d raw=%d silentDrop=%d", total, semantic, raw, silentDrop)
}

// TestUnknownVendorEventRawPassthrough proves a future/unknown vendor event is
// never dropped: it maps to "raw" with RawType preserved and a non-nil Raw payload.
func TestUnknownVendorEventRawPassthrough(t *testing.T) {
	raw := `{"directory":"/tmp","payload":{"type":"some.future.event","properties":{"foo":"bar"}}}`
	ev, err := opencode.MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if ev.Type != runtime.EventType("raw") {
		t.Fatalf("expected Type=raw, got %q", ev.Type)
	}
	if ev.RawType != "some.future.event" {
		t.Fatalf("expected RawType=some.future.event, got %q", ev.RawType)
	}
	if len(ev.Raw) == 0 {
		t.Fatal("Raw must not be empty for unknown event")
	}
}

// TestMalformedEventRawPassthrough proves unparseable JSON is not silently
// swallowed: it produces an _unparseable_ event that preserves the raw bytes.
func TestMalformedEventRawPassthrough(t *testing.T) {
	raw := `{broken json`
	ev, err := opencode.MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if ev.Type != runtime.EventType("_unparseable_") {
		t.Fatalf("expected Type=_unparseable_, got %q", ev.Type)
	}
	if string(ev.Raw) != raw {
		t.Fatalf("expected Raw to preserve exact bytes, got %q", ev.Raw)
	}
}
