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

func TestFilePartSourceVariantsSatisfyContract(t *testing.T) {
	var sources = []FilePartSource{FileSource{}, SymbolSource{}, ResourceSource{}}
	if len(sources) != 3 {
		t.Fatal("expected three file part source variants")
	}
}

func TestSensitivityValues(t *testing.T) {
	cases := map[Sensitivity]string{
		SensitivityPublic:    "public",
		SensitivityInternal:  "internal",
		SensitivitySensitive: "sensitive",
	}
	for s, want := range cases {
		if string(s) != want {
			t.Fatalf("%q != %q", s, want)
		}
	}
}
