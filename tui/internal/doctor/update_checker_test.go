package doctor

import (
	"context"
	"errors"
	"testing"
	"time"

	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/update"
)

type fakeCandidateFactory struct {
	rt     runtimedomain.AgentRuntime
	url    string
	err    error
	starts int
}

func (f *fakeCandidateFactory) Start(context.Context, update.Candidate) (runtimedomain.AgentRuntime, string, func(), error) {
	f.starts++
	return f.rt, f.url, func() {}, f.err
}

func TestUpdateCheckerUsesCandidateRuntimeAndDoctor(t *testing.T) {
	home, cfg := doctorInstall(t)
	root, _ := update.NewPlatformSwitcher(home).Current()
	rt := &fakeRuntime{
		health: runtimedomain.HealthInfo{Healthy: true, Version: "1.18.11"},
		agents: []runtimedomain.Agent{{Name: "code-reviewer"}, {Name: "unit-test-generator"}, {Name: "api-documentation"}},
	}
	factory := &fakeCandidateFactory{rt: rt, url: "http://127.0.0.1:4141"}
	checker := &UpdateChecker{Factory: factory, ExpectedOpenCodeVersion: "1.18.11", Timeout: time.Second}
	if err := checker.Check(context.Background(), update.CheckPreSwitch, update.Candidate{Version: "1.0.0", VersionDir: root, ConfigDir: cfg}); err != nil {
		t.Fatal(err)
	}
	if factory.starts != 1 {
		t.Fatalf("starts=%d, want 1", factory.starts)
	}
}

func TestUpdateCheckerFailsClosedWithoutFactory(t *testing.T) {
	checker := &UpdateChecker{}
	if err := checker.Check(context.Background(), update.CheckPreSwitch, update.Candidate{}); err == nil {
		t.Fatal("missing factory should fail")
	}
}

func TestUpdateCheckerPropagatesCandidateRuntimeFailure(t *testing.T) {
	checker := &UpdateChecker{Factory: &fakeCandidateFactory{err: errors.New("candidate runtime boom")}}
	if err := checker.Check(context.Background(), update.CheckPreSwitch, update.Candidate{}); err == nil {
		t.Fatal("candidate runtime failure should block update")
	}
}
