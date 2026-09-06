package app

import (
	"os"
	"strings"
	"testing"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func task32ProbeSuccess(sessionID runtime.SessionID, toolID, probe string) runtime.Event {
	return runtime.Event{
		Type:      eventTypeToolSuccess,
		SessionID: string(sessionID),
		Tool: &runtime.ToolEvent{
			Name: toolID,
			Metadata: map[string]string{
				"codeaProbe":        probe,
				"codeaProbeResult":  "pass",
				"codeaProbeAttempt": "1",
			},
		},
	}
}

func task32RunCmd(t *testing.T, m *Model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			task32RunCmd(t, m, child)
		}
		return
	}
	_, next := m.Update(msg)
	if next != nil {
		task32RunCmd(t, m, next)
	}
}

func TestTask32QualificationScoringIsConservative(t *testing.T) {
	tests := []struct{ name string; in ProbeOutcome; want modelprofile.CapabilityLevel }{
		{"first pass", ProbeOutcome{Attempts: 1, Passed: true}, modelprofile.CapabilityStrong},
		{"retry pass", ProbeOutcome{Attempts: 2, Passed: true}, modelprofile.CapabilityMedium},
		{"two failures", ProbeOutcome{Attempts: 2}, modelprofile.CapabilityWeak},
		{"missing", ProbeOutcome{}, modelprofile.CapabilityWeak},
		{"one failure at terminal", ProbeOutcome{Attempts: 1}, modelprofile.CapabilityWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := levelForProbe(tt.in); got != tt.want {
				t.Fatalf("levelForProbe(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTask32ModelCheckPromptUsesRegisteredProbeIDs(t *testing.T) {
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "coder-v2"}
	req := modelCheckPrompt(ref)
	if req.Agent != "model-evaluator" || req.Model == nil || *req.Model != ref {
		t.Fatalf("request identity = %#v", req)
	}
	if len(req.Parts) != 1 {
		t.Fatalf("parts = %d, want 1", len(req.Parts))
	}
	part, ok := req.Parts[0].(runtime.TextPart)
	if !ok || !part.Synthetic || part.Metadata["codea.kind"] != "model-check" || part.Text != fixedQualificationInstruction {
		t.Fatalf("qualification part = %#v", req.Parts[0])
	}
	for _, id := range []string{"probe_tool_call", "probe_structured", "probe_patch", "probe_plan"} {
		if !strings.Contains(part.Text, id) {
			t.Fatalf("qualification prompt missing registered tool id %q: %s", id, part.Text)
		}
	}
	for _, obsolete := range []string{"probe-state", "probe-structured", "probe-patch", "probe-plan"} {
		if strings.Contains(part.Text, obsolete) {
			t.Fatalf("qualification prompt still contains obsolete tool id %q: %s", obsolete, part.Text)
		}
	}
}

func TestTask32ModelCheckUpdateFlow(t *testing.T) {
	fake := fakeruntime.New()
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "coder"}
	fake.SetModels([]runtime.Model{{Ref: ref, Default: true}})
	m := NewModel(fake)
	m.SetModelProfileStore(modelprofile.NewStore(t.TempDir()))
	m.sessionID = runtime.SessionID("user-session")
	m.input = "/model-check"

	cmd := m.submit()
	if cmd == nil {
		t.Fatal("/model-check did not start asynchronous model resolution")
	}
	msg := cmd()
	_, cmd = m.Update(msg)
	if cmd == nil || m.modelCheck.Model != ref {
		t.Fatalf("Model.Update did not resolve exact model: state=%#v", m.modelCheck)
	}
	msg = cmd()
	_, cmd = m.Update(msg)
	if cmd == nil || m.modelCheck.SessionID == "" || m.modelCheck.SessionID == m.sessionID {
		t.Fatalf("Model.Update did not create isolated qualification session: %#v", m.modelCheck)
	}
	msg = cmd()
	_, _ = m.Update(msg)
	prompts := fake.Prompts()
	if len(prompts) != 1 || prompts[0].SessionID != m.modelCheck.SessionID || prompts[0].Request.Agent != "model-evaluator" {
		t.Fatalf("qualification prompt was not sent through real update flow: %#v", prompts)
	}
}

func TestTask32ModelCheckInternalEventInterceptAndStreamContinues(t *testing.T) {
	internal := runtime.SessionID("internal-check")
	m := NewModel(nil)
	m.sessionID = runtime.SessionID("user-session")
	m.modelCheck = ModelCheckState{Active: true, SessionID: internal, Model: runtime.ModelRef{ProviderID: "p", ModelID: "m"}}
	ch := make(chan runtime.Event, 1)
	m.eventCh = ch
	nextEvent := runtime.Event{Type: eventTypeToolCalled, SessionID: string(m.sessionID), Tool: &runtime.ToolEvent{Name: "user-tool"}}
	ch <- nextEvent

	_, cmd := m.Update(runtimeEventMsg{ev: task32ProbeSuccess(internal, "probe_tool_call", "tool_call")})
	if m.modelCheck.ToolCalling != (ProbeOutcome{Attempts: 1, Passed: true}) {
		t.Fatalf("internal event was not scored before user-session processing: %#v", m.modelCheck.ToolCalling)
	}
	if len(m.tools) != 0 {
		t.Fatalf("internal qualification event polluted visible tool state: %#v", m.tools)
	}
	if cmd == nil {
		t.Fatal("consumed internal event did not continue SSE wait")
	}
	msg := cmd()
	next, ok := msg.(runtimeEventMsg)
	if !ok || next.ev.Tool == nil || next.ev.Tool.Name != "user-tool" {
		t.Fatalf("SSE continuation did not wait for next event: %#v", msg)
	}
}

func TestTask32RealMetadataScoringAllFirstAttemptPassIsStrong(t *testing.T) {
	home := t.TempDir()
	store := modelprofile.NewStore(home)
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "strong"}
	internal := runtime.SessionID("internal-strong")
	m := NewModel(nil)
	m.SetModelProfileStore(store)
	m.modelCheck = ModelCheckState{Active: true, SessionID: internal, Model: ref}

	for _, probe := range []struct{ tool, metadata string }{
		{"probe_tool_call", "tool_call"},
		{"probe_structured", "structured"},
		{"probe_patch", "patch"},
		{"probe_plan", "planning"},
	} {
		_, _ = m.Update(runtimeEventMsg{ev: task32ProbeSuccess(internal, probe.tool, probe.metadata)})
	}
	_, cmd := m.Update(runtimeEventMsg{ev: runtime.Event{Type: eventTypeStepFinished, SessionID: string(internal)}})
	if cmd == nil {
		t.Fatal("step.finished did not schedule profile persistence")
	}
	task32RunCmd(t, m, cmd)
	profile, found, err := store.Load(ref)
	if err != nil || !found {
		t.Fatalf("strong profile not persisted: found=%t err=%v", found, err)
	}
	if profile.Overall != modelprofile.CapabilityStrong {
		t.Fatalf("real plugin metadata scored %q, want strong: %#v", profile.Overall, profile)
	}
}

func TestTask32SchemaFailureCountsAndRetryScoresMedium(t *testing.T) {
	home := t.TempDir()
	store := modelprofile.NewStore(home)
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "retry"}
	internal := runtime.SessionID("internal-retry")
	m := NewModel(nil)
	m.SetModelProfileStore(store)
	m.modelCheck = ModelCheckState{Active: true, SessionID: internal, Model: ref}

	_, _ = m.Update(runtimeEventMsg{ev: runtime.Event{Type: eventTypeToolFailed, SessionID: string(internal), Tool: &runtime.ToolEvent{Name: "probe_structured"}}})
	_, _ = m.Update(runtimeEventMsg{ev: task32ProbeSuccess(internal, "probe_structured", "structured")})
	for _, probe := range []struct{ tool, metadata string }{
		{"probe_tool_call", "tool_call"},
		{"probe_patch", "patch"},
		{"probe_plan", "planning"},
	} {
		_, _ = m.Update(runtimeEventMsg{ev: task32ProbeSuccess(internal, probe.tool, probe.metadata)})
	}
	if m.modelCheck.Structured != (ProbeOutcome{Attempts: 2, Passed: true}) {
		t.Fatalf("schema-invalid + valid retry was not mechanically counted: %#v", m.modelCheck.Structured)
	}
	_, cmd := m.Update(runtimeEventMsg{ev: runtime.Event{Type: eventTypeStepFinished, SessionID: string(internal)}})
	task32RunCmd(t, m, cmd)
	profile, found, err := store.Load(ref)
	if err != nil || !found {
		t.Fatalf("medium profile not persisted: found=%t err=%v", found, err)
	}
	if profile.StructuredOutput != modelprofile.CapabilityMedium || profile.Overall != modelprofile.CapabilityMedium {
		t.Fatalf("retry profile = %#v, want structured/overall medium", profile)
	}
}

func TestTask32ProbeAttemptsBoundedAtTwo(t *testing.T) {
	fake := fakeruntime.New()
	store := modelprofile.NewStore(t.TempDir())
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "weak"}
	internal := runtime.SessionID("internal-bounded")
	m := NewModel(fake)
	m.SetModelProfileStore(store)
	m.modelCheck = ModelCheckState{Active: true, SessionID: internal, Model: ref}
	failed := runtime.Event{Type: eventTypeToolFailed, SessionID: string(internal), Tool: &runtime.ToolEvent{Name: "probe_tool_call"}}

	_, _ = m.Update(runtimeEventMsg{ev: failed})
	_, cmd := m.Update(runtimeEventMsg{ev: failed})
	if cmd == nil || !m.modelCheck.Terminal || m.modelCheck.ToolCalling.Attempts != 2 {
		t.Fatalf("second failed attempt did not terminally bound probe: %#v", m.modelCheck)
	}
	_, _ = m.Update(runtimeEventMsg{ev: failed})
	if m.modelCheck.ToolCalling.Attempts != 2 {
		t.Fatalf("third attempt changed bounded state: %#v", m.modelCheck.ToolCalling)
	}
	task32RunCmd(t, m, cmd)
	cancelled := fake.CancelledSessions()
	if len(cancelled) != 1 || cancelled[0] != internal {
		t.Fatalf("qualification session was not cancelled at attempt bound: %#v", cancelled)
	}
	profile, found, err := store.Load(ref)
	if err != nil || !found || profile.Overall != modelprofile.CapabilityWeak {
		t.Fatalf("two failures did not persist weak result: profile=%#v found=%t err=%v", profile, found, err)
	}
}

func TestTask32AbortFallbackThroughModelUpdate(t *testing.T) {
	home := t.TempDir()
	store := modelprofile.NewStore(home)
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "aborted"}
	internal := runtime.SessionID("internal-aborted")
	m := NewModel(nil)
	m.SetModelProfileStore(store)
	m.runtimeModels = []runtime.Model{{Ref: ref, Default: true}}
	m.modelCheck = ModelCheckState{Active: true, SessionID: internal, Model: ref}

	_, _ = m.Update(runtimeEventMsg{ev: runtime.Event{Type: eventTypeRuntimeError, SessionID: string(internal), Error: &runtime.RuntimeError{Kind: runtime.RuntimeErrorCancelled, Message: "cancelled"}}})
	if m.modelCheck.Active || m.modelCheck.Terminal {
		t.Fatalf("runtime abort left qualification active/terminal: %#v", m.modelCheck)
	}
	if _, err := os.Stat(modelprofile.ProfilePath(home, ref)); !os.IsNotExist(err) {
		t.Fatalf("aborted qualification persisted a completed profile: %v", err)
	}
	if got := m.currentModelStrategy(runtime.PromptRequest{}).Level; got != modelprofile.CapabilityMedium {
		t.Fatalf("abort fallback = %q, want medium", got)
	}
}

func TestTask32InternalSessionHiddenThroughModelUpdate(t *testing.T) {
	m := NewModel(nil)
	m.sessionID = runtime.SessionID("user-1")
	_, _ = m.Update(listSessionsResultMsg{sessions: []runtime.Session{
		{ID: "user-1", Title: "Feature work"},
		{ID: "internal-1", Title: internalModelCheckTitlePrefix + "private/coder"},
		{ID: "user-2", Title: "Review"},
	}})
	if !m.sessionPanel.Visible || len(m.sessionPanel.Items) != 2 {
		t.Fatalf("session picker did not filter through Model.Update: %#v", m.sessionPanel)
	}
	for _, item := range m.sessionPanel.Items {
		if strings.HasPrefix(item.Title, internalModelCheckTitlePrefix) {
			t.Fatalf("internal model-check session leaked into picker: %#v", item)
		}
	}
}

func TestTask32UniqueRuntimeDefaultResolution(t *testing.T) {
	models := []runtime.Model{{Ref: runtime.ModelRef{ProviderID: "p", ModelID: "a"}, Default: true}, {Ref: runtime.ModelRef{ProviderID: "p", ModelID: "b"}}}
	ref, ok := uniqueDefaultModel(models)
	if !ok || ref.ModelID != "a" {
		t.Fatalf("unique default = %#v, %t", ref, ok)
	}
	models[1].Default = true
	if _, ok := uniqueDefaultModel(models); ok {
		t.Fatal("multiple defaults must be ambiguous")
	}
}
