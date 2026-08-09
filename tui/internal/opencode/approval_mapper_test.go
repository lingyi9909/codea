package opencode

import (
	"encoding/json"
	"testing"

	"codea/tui/internal/runtime"
)

func TestMapApprovalReplyOnce(t *testing.T) {
	got := MapApprovalReply(runtime.ApprovalReply{
		Decision: runtime.ApprovalOnce,
	})
	if got.Reply != "once" {
		t.Fatalf("expected Reply=once, got %q", got.Reply)
	}
}

func TestMapApprovalReplyAlways(t *testing.T) {
	got := MapApprovalReply(runtime.ApprovalReply{
		Decision: runtime.ApprovalAlways,
	})
	if got.Reply != "always" {
		t.Fatalf("expected Reply=always, got %q", got.Reply)
	}
}

func TestMapApprovalReplyReject(t *testing.T) {
	got := MapApprovalReply(runtime.ApprovalReply{
		Decision: runtime.ApprovalReject,
	})
	if got.Reply != "reject" {
		t.Fatalf("expected Reply=reject, got %q", got.Reply)
	}
}

func TestMapApprovalReplyWithMessage(t *testing.T) {
	got := MapApprovalReply(runtime.ApprovalReply{
		Decision: runtime.ApprovalReject,
		Message:  "denied by user",
	})
	if got.Reply != "reject" || got.Message != "denied by user" {
		t.Fatalf("unexpected reply: %+v", got)
	}
}

func TestMapApprovalReplyNoRememberField(t *testing.T) {
	got := MapApprovalReply(runtime.ApprovalReply{
		Decision: runtime.ApprovalOnce,
		Message:  "ok",
	})
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if _, exists := m["remember"]; exists {
		t.Fatal("approval reply must not contain remember field")
	}
}
