package app

import (
    "context"
    "strings"
    "testing"

    tea "github.com/charmbracelet/bubbletea"

    "codea/tui/internal/doctor"
    "codea/tui/internal/runtime"
    fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

type task23DoctorCheck struct{}

func (task23DoctorCheck) Run(context.Context) doctor.Result {
    return doctor.Result{Name: "Task23 Doctor", Category: doctor.CategoryBehavior, Status: doctor.Pass, Detail: "shared-service"}
}

func TestTask23ModelSelectionIsSessionScopedAndAppliedToPrompt(t *testing.T) {
    fake := fakeruntime.New()
    fake.SetModels([]runtime.Model{
        {Ref: runtime.ModelRef{ProviderID: "company", ModelID: "kimi"}, Name: "Kimi", ProviderName: "Company AI", Default: true},
        {Ref: runtime.ModelRef{ProviderID: "company", ModelID: "coder"}, Name: "Coder", ProviderName: "Company AI"},
    })
    m := NewModel(fake)
    m.sessionID = runtime.SessionID("s1")
    m.input = "/model"

    cmd := m.submit()
    if cmd == nil { t.Fatal("/model should list runtime models asynchronously") }
    msg := cmd()
    _, _ = m.Update(msg)
    if !m.modelPicker.Visible || len(m.modelPicker.Items) != 2 {
        t.Fatalf("model picker = %#v", m.modelPicker)
    }

    _ = m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
    _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
    selected, ok := m.sessionModels[runtime.SessionID("s1")]
    if !ok || selected.ModelID != "coder" {
        t.Fatalf("selected model = %#v, ok=%t", selected, ok)
    }

    m.input = "hello"
    cmd = m.submit()
    if cmd == nil { t.Fatal("prompt command missing") }
    _ = cmd()
    prompts := fake.Prompts()
    if len(prompts) != 1 || prompts[0].Request.Model == nil || prompts[0].Request.Model.ModelID != "coder" {
        t.Fatalf("prompt model = %#v", prompts)
    }

    m.sessionID = runtime.SessionID("s2")
    m.isStreaming = false
    m.input = "new session"
    cmd = m.submit()
    _ = cmd()
    prompts = fake.Prompts()
    if len(prompts) != 2 || prompts[1].Request.Model != nil {
        t.Fatalf("new session inherited explicit model: %#v", prompts[1].Request.Model)
    }
}

func TestTask23CompactFailsClosedWhenCapabilityMissing(t *testing.T) {
    fake := fakeruntime.New()
    fake.CapabilitiesConfig.ContextCompaction = false
    m := NewModel(fake)
    m.sessionID = runtime.SessionID("s1")
    m.input = "/compact"

    if cmd := m.submit(); cmd != nil {
        t.Fatal("unsupported compact must fail closed synchronously")
    }
    if fake.CompactCalls() != 0 {
        t.Fatalf("compact calls = %d, want 0", fake.CompactCalls())
    }
    if len(m.messages) == 0 || !strings.Contains(strings.ToLower(m.messages[len(m.messages)-1].Content), "unsupported") {
        t.Fatalf("missing explicit unsupported message: %#v", m.messages)
    }
}

func TestTask23CompactUsesCurrentSession(t *testing.T) {
    fake := fakeruntime.New()
    fake.CapabilitiesConfig.ContextCompaction = true
    m := NewModel(fake)
    m.sessionID = runtime.SessionID("s1")
    m.input = "/compact"

    cmd := m.submit()
    if cmd == nil { t.Fatal("compact command missing") }
    msg := cmd()
    _, _ = m.Update(msg)
    ids := fake.CompactedSessions()
    if len(ids) != 1 || ids[0] != runtime.SessionID("s1") {
        t.Fatalf("compacted sessions = %#v", ids)
    }
}

func TestTask23StatusIsUsefulAndSanitized(t *testing.T) {
    fake := fakeruntime.New()
    fake.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "1.18.11"}
    fake.CapabilitiesConfig.Streaming = true
    fake.CapabilitiesConfig.Reasoning = true
    fake.CapabilitiesConfig.ToolApproval = true
    fake.CapabilitiesConfig.ContextCompaction = true
    m := NewModel(fake)
    m.SetWorkspaceInfo(WorkspaceInfo{CodeaVersion: "1.1-dev", RuntimeProvider: "OpenCode", Project: `C:\repo`})
    m.sessionID = runtime.SessionID("s1")
    m.sessionModels[m.sessionID] = runtime.ModelRef{ProviderID: "company", ModelID: "kimi"}
    m.loadedSkillIDs = []string{"review-skill"}
    m.input = "/status"

    cmd := m.submit()
    if cmd == nil { t.Fatal("status should query runtime health") }
    msg := cmd()
    _, _ = m.Update(msg)
    got := m.messages[len(m.messages)-1].Content
    for _, want := range []string{"1.1-dev", "OpenCode", "1.18.11", "s1", "company/kimi", "review-skill", "Compaction: true"} {
        if !strings.Contains(got, want) { t.Fatalf("status missing %q:\n%s", want, got) }
    }
    if strings.Contains(strings.ToLower(got), "apikey") || strings.Contains(got, "secret") {
        t.Fatalf("status leaked sensitive config: %s", got)
    }
}

func TestTask23DoctorUsesSharedDoctorService(t *testing.T) {
    fake := fakeruntime.New()
    m := NewModel(fake)
    m.SetDoctorService(doctor.NewService(task23DoctorCheck{}))
    m.input = "/doctor"

    cmd := m.submit()
    if cmd == nil { t.Fatal("doctor command missing") }
    msg := cmd()
    _, _ = m.Update(msg)
    got := m.messages[len(m.messages)-1].Content
    if !strings.Contains(got, "Task23 Doctor") || !strings.Contains(got, "PASS") {
        t.Fatalf("doctor output = %q", got)
    }
}
