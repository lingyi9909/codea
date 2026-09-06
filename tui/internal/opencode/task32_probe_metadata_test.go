package opencode

import "testing"

func TestTask32ProbeMetadataAllowlist(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"internal-model-check","part":{"id":"p1","messageID":"m1","sessionID":"internal-model-check","type":"tool","tool":"probe_structured","callID":"c1","state":{"status":"completed","input":{"secret":"must-not-cross"},"metadata":{"codeaProbe":"structured","codeaProbeResult":"pass","codeaProbeAttempt":"2","vendorSecret":"must-not-cross","rawArgs":"must-not-cross","rawOutput":"must-not-cross","codeaPlugin":"codea-enterprise"}}}}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.Tool == nil {
		t.Fatal("expected tool event")
	}
	want := map[string]string{
		"codeaProbe":        "structured",
		"codeaProbeResult":  "pass",
		"codeaProbeAttempt": "2",
	}
	if len(event.Tool.Metadata) != len(want) {
		t.Fatalf("tool metadata = %#v, want exactly %#v", event.Tool.Metadata, want)
	}
	for key, value := range want {
		if event.Tool.Metadata[key] != value {
			t.Fatalf("tool metadata[%q] = %q, want %q", key, event.Tool.Metadata[key], value)
		}
	}
	for _, forbidden := range []string{"vendorSecret", "rawArgs", "rawOutput", "codeaPlugin", "secret"} {
		if _, ok := event.Tool.Metadata[forbidden]; ok {
			t.Fatalf("forbidden vendor metadata crossed runtime boundary: %s", forbidden)
		}
	}
}

func TestTask32ProbeMetadataRejectsUnboundedValues(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"internal-model-check","part":{"id":"p1","messageID":"m1","sessionID":"internal-model-check","type":"tool","tool":"probe_plan","callID":"c1","state":{"status":"completed","metadata":{"codeaProbe":"planning","codeaProbeResult":"maybe","codeaProbeAttempt":"99"}}}}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.Tool == nil {
		t.Fatal("expected tool event")
	}
	if len(event.Tool.Metadata) != 0 {
		t.Fatalf("unbounded probe metadata must be dropped, got %#v", event.Tool.Metadata)
	}
}
