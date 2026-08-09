package runtime

import (
	"testing"
)

func TestApprovalDecisionValues(t *testing.T) {
	cases := map[ApprovalDecision]string{
		ApprovalOnce:   "once",
		ApprovalAlways: "always",
		ApprovalReject: "reject",
	}
	for decision, want := range cases {
		if string(decision) != want {
			t.Fatalf("%q != %q", decision, want)
		}
	}
}

func TestApprovalReplyCarriesDecisionAndOptionalMessage(t *testing.T) {
	reply := ApprovalReply{Decision: ApprovalReject, Message: "denied by user"}
	if reply.Decision != ApprovalReject || reply.Message != "denied by user" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestPromptPartVariantsSatisfyContract(t *testing.T) {
	var parts = []PromptPart{TextPart{}, FilePart{}, AgentPart{}, SubtaskPart{}}
	if len(parts) != 4 {
		t.Fatal("expected four prompt part variants")
	}
}
