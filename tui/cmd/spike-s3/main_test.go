package main

import (
	"strings"
	"testing"
)

func TestReadUntilPermissionReturnsTargetSessionRequest(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"directory":"/tmp/other","payload":{"type":"permission.asked","properties":{"id":"per_other","sessionID":"ses_other","permission":"bash","patterns":["echo other"],"metadata":{},"always":[]}}}`,
		``,
		`data: {"directory":"/tmp/project","payload":{"type":"permission.asked","properties":{"id":"per_target","sessionID":"ses_target","permission":"bash","patterns":["touch approved.txt"],"metadata":{"command":"touch approved.txt"},"always":["touch *"]}}}`,
		``,
	}, "\n")

	request, events, err := readUntilPermission(strings.NewReader(stream), "ses_target")
	if err != nil {
		t.Fatalf("readUntilPermission returned error: %v", err)
	}
	if request.ID != "per_target" || request.SessionID != "ses_target" {
		t.Fatalf("got request %#v, want target permission", request)
	}
	if request.Permission != "bash" || len(request.Patterns) != 1 || request.Patterns[0] != "touch approved.txt" {
		t.Fatalf("unexpected permission details: %#v", request)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
}

func TestReadUntilIdleRejectsSessionError(t *testing.T) {
	stream := `data: {"directory":"/tmp/project","payload":{"type":"session.error","properties":{"sessionID":"ses_target","error":{"name":"UnknownError","data":{"message":"tool failed"}}}}}` + "\n\n"

	_, err := readUntilIdle(strings.NewReader(stream), "ses_target")
	if err == nil || !strings.Contains(err.Error(), "session.error") {
		t.Fatalf("got error %v, want session.error", err)
	}
}
