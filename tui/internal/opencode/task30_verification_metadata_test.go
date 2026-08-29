package opencode

import "testing"

func TestVerificationToolMetadataAllowlist(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"turn-1","sessionID":"s1","type":"tool","tool":"verify_project","callID":"verify-1","state":{"status":"completed","metadata":{"codeaPlugin":"codea-enterprise","codeaVerification":"true","codeaVerificationResult":"pass","codeaVerificationProfile":"go","secret":"must-not-cross","rawOutput":"must-not-cross","command":"must-not-cross"}}}}}}`)
	ev, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}
	if ev.Tool == nil {
		t.Fatal("expected tool event")
	}
	want := map[string]string{
		"codeaVerification":        "true",
		"codeaVerificationResult":  "pass",
		"codeaVerificationProfile": "go",
	}
	if len(ev.Tool.Metadata) != len(want) {
		t.Fatalf("metadata=%#v, want only safe verification fields", ev.Tool.Metadata)
	}
	for key, value := range want {
		if ev.Tool.Metadata[key] != value {
			t.Fatalf("metadata[%q]=%q, want %q", key, ev.Tool.Metadata[key], value)
		}
	}
	for _, key := range []string{"secret", "rawOutput", "command", "codeaPlugin"} {
		if _, ok := ev.Tool.Metadata[key]; ok {
			t.Fatalf("unallowlisted vendor metadata crossed Runtime boundary: %s", key)
		}
	}
}
