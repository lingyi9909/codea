package app

import (
	"strings"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func approvalEvent(session string) runtime.Event {
	return approvalEventID(session, "per_1")
}

func approvalEventID(session, id string) runtime.Event {
	return runtime.Event{
		Type:      eventTypeApprovalRequested,
		SessionID: session,
		Approval:  &runtime.ApprovalRequest{ID: id, Permission: "shell", Command: "rm -rf ./build"},
	}
}

func TestApprovalCurrentSessionAllowOnce(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})
	if !m.permission.Visible() {
		t.Fatal("current-session approval should open the modal")
	}

	_, cmd := m.Update(yKey())
	if cmd == nil {
		t.Fatal("allow-once should issue ReplyApprovalCmd")
	}
	ar, ok := cmd().(approvalResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want approvalResultMsg", cmd())
	}
	if ar.err != nil {
		t.Fatalf("ReplyApproval error: %v", ar.err)
	}

	approvals := client.Approvals()
	if len(approvals) != 1 {
		t.Fatalf("approvals = %d, want 1", len(approvals))
	}
	if approvals[0].ID != runtime.ApprovalID("per_1") {
		t.Errorf("approval ID = %q, want per_1", approvals[0].ID)
	}
	if approvals[0].Reply.Decision != runtime.ApprovalOnce {
		t.Errorf("decision = %q, want once", approvals[0].Reply.Decision)
	}

	m.Update(ar)
	if m.permission.Visible() {
		t.Error("modal should close after a successful reply")
	}
}

func TestApprovalForeignSessionIgnored(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")

	m.Update(runtimeEventMsg{ev: approvalEvent("s2")})

	if m.permission.Visible() {
		t.Error("foreign-session approval must not open the modal")
	}
}

func TestApprovalAfterResumeUsesNewSession(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})
	if m.permission.Visible() {
		t.Error("old-session approval after resume must not open the modal")
	}

	m.Update(runtimeEventMsg{ev: approvalEvent("s2")})
	if !m.permission.Visible() {
		t.Error("resumed-session approval should open the modal")
	}
}

func TestApprovalAlwaysDecision(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")
	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})

	_, cmd := m.Update(aKey())
	if cmd == nil {
		t.Fatal("always should issue ReplyApprovalCmd")
	}
	cmd()

	approvals := client.Approvals()
	if len(approvals) != 1 {
		t.Fatalf("approvals = %d, want 1", len(approvals))
	}
	if approvals[0].Reply.Decision != runtime.ApprovalAlways {
		t.Errorf("decision = %q, want always", approvals[0].Reply.Decision)
	}
}

func TestApprovalRejectDecision(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")
	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})

	_, cmd := m.Update(nKey())
	if cmd == nil {
		t.Fatal("reject should issue ReplyApprovalCmd")
	}
	cmd()

	approvals := client.Approvals()
	if len(approvals) != 1 {
		t.Fatalf("approvals = %d, want 1", len(approvals))
	}
	if approvals[0].Reply.Decision != runtime.ApprovalReject {
		t.Errorf("decision = %q, want reject", approvals[0].Reply.Decision)
	}
}

func TestApprovalRejectViaEsc(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")
	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})

	_, cmd := m.Update(escKey())
	if cmd == nil {
		t.Fatal("esc should reject and issue ReplyApprovalCmd")
	}
	cmd()

	approvals := client.Approvals()
	if len(approvals) != 1 || approvals[0].Reply.Decision != runtime.ApprovalReject {
		t.Errorf("esc should map to reject, got %+v", approvals)
	}
}

func TestApprovalErrorKeepsModalOpen(t *testing.T) {
	client := fakeruntime.New()
	client.ReplyApprovalError = fakeruntime.ErrSimulated
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")
	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})

	_, cmd := m.Update(yKey())
	ar, ok := cmd().(approvalResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want approvalResultMsg", cmd())
	}
	if ar.err == nil {
		t.Fatal("expected ReplyApproval error")
	}

	m.Update(ar)
	if !m.permission.Visible() {
		t.Error("modal should stay open on error so the user can retry/reject")
	}
	if m.approvalErr == "" {
		t.Error("approvalErr should be set on failure")
	}
}

func TestApprovalModalBlocksChatKeys(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})

	m.input = ""
	m.Update(runeKey("x"))
	if m.input != "" {
		t.Errorf("input = %q, want empty (typing must not leak into chat during approval)", m.input)
	}

	_, cmd := m.Update(enterKey())
	if cmd != nil {
		t.Error("enter must not submit while approval modal is open")
	}

	m.Update(ctrlSKey())
	if m.sessionPanel.Visible {
		t.Error("session shortcut must not open the panel while approval modal is open")
	}
}

func TestApprovalViewRendersDangerWarning(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.width = 80
	m.height = 24
	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})

	out := m.View()
	for _, want := range []string{"Tool approval required", "shell", "rm -rf ./build", "Potentially dangerous command"} {
		if !strings.Contains(out, want) {
			t.Errorf("approval view missing %q", want)
		}
	}
}

func TestApprovalStaleSuccessDoesNotCloseNewRequest(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	m.Update(runtimeEventMsg{ev: approvalEventID("s1", "per_1")})
	_, cmd := m.Update(yKey())
	if cmd == nil {
		t.Fatal("expected ReplyApprovalCmd for A")
	}

	// B arrives while A's reply is still in flight.
	m.Update(runtimeEventMsg{ev: approvalEventID("s1", "per_2")})
	if m.permission.Request == nil || m.permission.Request.ID != "per_2" {
		t.Fatalf("modal should show B after it arrives, got %+v", m.permission.Request)
	}

	// A's result returns (success). It must not close B.
	ar, ok := cmd().(approvalResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want approvalResultMsg", cmd())
	}
	if ar.err != nil {
		t.Fatalf("A reply error: %v", ar.err)
	}
	m.Update(ar)

	if !m.permission.Visible() {
		t.Error("B must remain visible after A's stale success result")
	}
	if m.permission.Request.ID != "per_2" {
		t.Errorf("modal = %q, want B (per_2)", m.permission.Request.ID)
	}
}

func TestApprovalStaleErrorDoesNotPolluteNewRequest(t *testing.T) {
	client := fakeruntime.New()
	client.ReplyApprovalError = fakeruntime.ErrSimulated
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	m.Update(runtimeEventMsg{ev: approvalEventID("s1", "per_1")})
	_, cmd := m.Update(yKey())

	m.Update(runtimeEventMsg{ev: approvalEventID("s1", "per_2")})

	ar, ok := cmd().(approvalResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want approvalResultMsg", cmd())
	}
	if ar.err == nil {
		t.Fatal("expected A reply error")
	}
	m.Update(ar)

	if !m.permission.Visible() {
		t.Error("B must remain visible after A's stale error result")
	}
	if m.permission.Request.ID != "per_2" {
		t.Errorf("modal = %q, want B (per_2)", m.permission.Request.ID)
	}
	if m.approvalErr != "" {
		t.Errorf("approvalErr = %q, want empty (A's error must not pollute B)", m.approvalErr)
	}
}

func TestApprovalDuplicateKeyIgnoredWhilePending(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	m.Update(runtimeEventMsg{ev: approvalEvent("s1")})
	_, cmd := m.Update(yKey())
	if cmd == nil {
		t.Fatal("expected first ReplyApprovalCmd")
	}

	// While the reply is in flight, further decisions must be swallowed.
	for _, k := range []tea.KeyMsg{aKey(), nKey(), yKey(), escKey()} {
		if _, c := m.Update(k); c != nil {
			t.Errorf("%v while pending must not issue a second ReplyApprovalCmd", k)
		}
	}

	ar, ok := cmd().(approvalResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want approvalResultMsg", cmd())
	}
	m.Update(ar)

	if got := len(client.Approvals()); got != 1 {
		t.Errorf("approvals = %d, want exactly 1 (duplicate keys must be swallowed)", got)
	}
}
