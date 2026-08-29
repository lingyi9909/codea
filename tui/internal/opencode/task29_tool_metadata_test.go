package opencode

import "testing"

func TestTaskPlanToolMetadataAllowlist(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"p1","messageID":"turn-1","sessionID":"s1","type":"tool","tool":"task_step","callID":"c1","state":{"status":"completed","metadata":{"codeaPlugin":"codea-enterprise","codeaTaskPlan":"true","codeaPlanTotal":"5","codeaPlanCompleted":"2","codeaPlanActive":"step-3","secret":"must-not-cross","rawOutput":"must-not-cross"}}}}}}`)
	ev, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}
	if ev.Tool == nil {
		t.Fatal("expected tool event")
	}
	want := map[string]string{
		"codeaTaskPlan": "true",
		"codeaPlanTotal": "5",
		"codeaPlanCompleted": "2",
		"codeaPlanActive": "step-3",
	}
	if len(ev.Tool.Metadata) != len(want) {
		t.Fatalf("metadata=%#v, want only safe planning fields", ev.Tool.Metadata)
	}
	for key, value := range want {
		if ev.Tool.Metadata[key] != value {
			t.Fatalf("metadata[%q]=%q, want %q", key, ev.Tool.Metadata[key], value)
		}
	}
	if _, ok := ev.Tool.Metadata["secret"]; ok {
		t.Fatal("unallowlisted vendor metadata crossed Runtime boundary")
	}
}
