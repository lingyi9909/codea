package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"codea/tui/internal/repoctx"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

type fakeRepoContextService struct {
	result repoctx.RepoMap
	err    error
	called chan struct{}
	block  <-chan struct{}
}

func (f *fakeRepoContextService) BuildMap(ctx context.Context, q repoctx.Query) (repoctx.RepoMap, error) {
	if f.called != nil {
		select {
		case <-f.called:
		default:
			close(f.called)
		}
	}
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return repoctx.RepoMap{}, ctx.Err()
		}
	}
	return f.result, f.err
}

func completeRepoContextStage(t *testing.T, m *Model, cmd func() interface{}) {
	t.Helper()
	_ = m
	_ = cmd
}

func repoMapFixture() repoctx.RepoMap {
	return repoctx.RepoMap{
		Summary: "task relevant repository context",
		Files:   []string{"src/OrderService.java"},
		Symbols: []repoctx.Symbol{{ID: "order-service", Path: "src/OrderService.java", Name: "OrderService", StartLine: 1}},
	}
}

func assertRepoPromptParts(t *testing.T, req runtime.PromptRequest, userText string) {
	t.Helper()
	if len(req.Parts) != 2 {
		t.Fatalf("parts=%#v, want synthetic repo map + original text", req.Parts)
	}
	repoPart, ok := req.Parts[0].(runtime.TextPart)
	if !ok {
		t.Fatalf("parts[0] type=%T, want runtime.TextPart", req.Parts[0])
	}
	if !repoPart.Synthetic {
		t.Fatal("repo map part must be synthetic")
	}
	if repoPart.Metadata["codea.kind"] != "repo-map" {
		t.Fatalf("repo map metadata=%#v", repoPart.Metadata)
	}
	if !strings.Contains(repoPart.Text, "REPO CONTEXT") || !strings.Contains(repoPart.Text, "src/OrderService.java") {
		t.Fatalf("unexpected repo map text: %q", repoPart.Text)
	}
	userPart, ok := req.Parts[1].(runtime.TextPart)
	if !ok {
		t.Fatalf("parts[1] type=%T, want runtime.TextPart", req.Parts[1])
	}
	if userPart.Synthetic {
		t.Fatal("original prompt must remain non-synthetic")
	}
	if userPart.Text != userText {
		t.Fatalf("original prompt changed: got=%q want=%q", userPart.Text, userText)
	}
}

func TestRepoContextNormalPromptPrependsSyntheticMap(t *testing.T) {
	fakeRuntime := fakeruntime.New()
	m := NewModel(fakeRuntime)
	m.sessionID = runtime.SessionID("active")
	m.SetRepoContextService(&fakeRepoContextService{result: repoMapFixture()})

	cmd := m.startPrompt("Fix OrderService", "Fix OrderService", "general")
	if cmd == nil {
		t.Fatal("startPrompt should return asynchronous repo-context command")
	}
	msg := cmd()
	_, next := m.Update(msg)
	if next == nil {
		t.Fatal("repo-context result should continue to Runtime prompt")
	}
	_ = next()

	prompts := fakeRuntime.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("Runtime prompts=%d, want 1", len(prompts))
	}
	assertRepoPromptParts(t, prompts[0].Request, "Fix OrderService")
}

func TestRepoContextProfessionalPromptPreservesExpandedPrompt(t *testing.T) {
	fakeRuntime := fakeruntime.New()
	m := NewModel(fakeRuntime)
	m.sessionID = runtime.SessionID("active")
	m.SetRepoContextService(&fakeRepoContextService{result: repoMapFixture()})

	promptText := "Review order implementation:\nOrderService --changed-only"
	cmd := m.startCommandPrompt("/check-order OrderService --changed-only", promptText, "code-reviewer")
	msg := cmd()
	_, next := m.Update(msg)
	if next == nil {
		t.Fatal("repo-context result should continue professional prompt")
	}
	_ = next()

	prompts := fakeRuntime.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("Runtime prompts=%d, want 1", len(prompts))
	}
	if prompts[0].Request.Agent != "code-reviewer" {
		t.Fatalf("agent=%q", prompts[0].Request.Agent)
	}
	assertRepoPromptParts(t, prompts[0].Request, promptText)
}

func TestRepoContextSubmitDoesNotSynchronouslyIndex(t *testing.T) {
	m := NewModel(fakeruntime.New())
	called := make(chan struct{})
	release := make(chan struct{})
	m.SetRepoContextService(&fakeRepoContextService{result: repoMapFixture(), called: called, block: release})
	m.input = "Fix OrderService"

	cmd := m.submit()
	if cmd == nil {
		t.Fatal("submit should return async command")
	}
	select {
	case <-called:
		t.Fatal("BuildMap ran synchronously inside submit/Update path")
	default:
	}
	close(release)
}

func TestRepoContextFailureDegradesToOriginalPrompt(t *testing.T) {
	fakeRuntime := fakeruntime.New()
	m := NewModel(fakeRuntime)
	m.sessionID = runtime.SessionID("active")
	m.SetRepoContextService(&fakeRepoContextService{err: errors.New("index unavailable")})

	cmd := m.startPrompt("Fix OrderService", "Fix OrderService", "general")
	msg := cmd()
	_, next := m.Update(msg)
	if next == nil {
		t.Fatal("repo-context failure must not block prompt")
	}
	_ = next()
	prompts := fakeRuntime.Prompts()
	if len(prompts) != 1 || len(prompts[0].Request.Parts) != 1 {
		t.Fatalf("degraded prompt=%#v", prompts)
	}
	part, ok := prompts[0].Request.Parts[0].(runtime.TextPart)
	if !ok || part.Text != "Fix OrderService" || part.Synthetic {
		t.Fatalf("degraded user part=%#v", prompts[0].Request.Parts[0])
	}
	foundNotice := false
	for _, message := range m.messages {
		if message.Role == RoleInfo && strings.Contains(message.Content, "Repo Context unavailable") {
			foundNotice = true
		}
	}
	if !foundNotice {
		t.Fatalf("missing repo-context degradation notice: %#v", m.messages)
	}
}

func TestRepoContextFirstPromptPreservesCreateSessionOrdering(t *testing.T) {
	fakeRuntime := fakeruntime.New()
	m := NewModel(fakeRuntime)
	m.SetRepoContextService(&fakeRepoContextService{result: repoMapFixture()})

	cmd := m.startPrompt("Fix OrderService", "Fix OrderService", "general")
	repoMsg := cmd()
	_, createCmd := m.Update(repoMsg)
	if createCmd == nil {
		t.Fatal("prepared first prompt should create session before prompting")
	}
	sessionMsg := createCmd()
	_, promptCmd := m.Update(sessionMsg)
	if promptCmd == nil {
		t.Fatal("session creation should release pending prompt")
	}
	_ = promptCmd()

	prompts := fakeRuntime.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("Runtime prompts=%d, want 1", len(prompts))
	}
	if prompts[0].SessionID == "" {
		t.Fatal("prompt sent before session identity was established")
	}
	assertRepoPromptParts(t, prompts[0].Request, "Fix OrderService")
}
