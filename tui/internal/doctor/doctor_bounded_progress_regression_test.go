package doctor

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	runtimedomain "codea/tui/internal/runtime"
)

type boundedDoctorRuntime struct {
	subscribeCalls atomic.Int32
	events         chan runtimedomain.Event
	blockSSE       bool
	blockCancel    bool
}

func (r *boundedDoctorRuntime) Health(context.Context) (runtimedomain.HealthInfo, error) {
	return runtimedomain.HealthInfo{Healthy: true, Version: "1.18.11"}, nil
}
func (r *boundedDoctorRuntime) CreateSession(context.Context, runtimedomain.CreateSessionRequest) (runtimedomain.Session, error) {
	return runtimedomain.Session{ID: "doctor-session"}, nil
}
func (r *boundedDoctorRuntime) Prompt(_ context.Context, id runtimedomain.SessionID, _ runtimedomain.PromptRequest) error {
	if r.blockCancel {
		return errors.New("model unavailable")
	}
	if r.events != nil {
		r.events <- runtimedomain.Event{SessionID: string(id), Content: "CODEA_DOCTOR_OK"}
	}
	return nil
}
func (r *boundedDoctorRuntime) Subscribe(ctx context.Context) (<-chan runtimedomain.Event, error) {
	call := r.subscribeCalls.Add(1)
	if r.blockSSE && call >= 2 {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if r.events == nil {
		r.events = make(chan runtimedomain.Event, 4)
	}
	return r.events, nil
}
func (r *boundedDoctorRuntime) ReplyApproval(context.Context, runtimedomain.ApprovalID, runtimedomain.ApprovalReply) error {
	return nil
}
func (r *boundedDoctorRuntime) Cancel(ctx context.Context, _ runtimedomain.SessionID) error {
	if r.blockCancel {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}
func (r *boundedDoctorRuntime) ListAgents(context.Context) ([]runtimedomain.Agent, error) {
	return []runtimedomain.Agent{{Name: "code-reviewer"}, {Name: "unit-test-generator"}, {Name: "api-documentation"}}, nil
}
func (r *boundedDoctorRuntime) ListModels(context.Context) ([]runtimedomain.Model, error) { return nil, nil }
func (r *boundedDoctorRuntime) ListSessions(context.Context) ([]runtimedomain.Session, error) { return nil, nil }
func (r *boundedDoctorRuntime) GetSessionMessages(context.Context, runtimedomain.SessionID) ([]runtimedomain.Message, error) {
	return nil, nil
}
func (r *boundedDoctorRuntime) CompactSession(context.Context, runtimedomain.SessionID) error { return nil }
func (r *boundedDoctorRuntime) Capabilities() runtimedomain.RuntimeCapabilities {
	return runtimedomain.RuntimeCapabilities{}
}

func TestDoctorSSECheckCannotHangPastBehaviorTimeout(t *testing.T) {
	home, cfg := doctorInstall(t)
	rt := &boundedDoctorRuntime{blockSSE: true}
	svc, err := NewDefaultService(Config{
		HomeDir: home, ConfigDir: cfg, Runtime: rt,
		RuntimeURL: "http://127.0.0.1:4141", ExpectedOpenCodeVersion: "1.18.11",
		BehaviorTimeout: 25 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan Report, 1)
	go func() { done <- svc.Run(context.Background()) }()
	select {
	case report := <-done:
		res := resultByName(t, report, "SSE")
		if res.Status != Fail || !strings.Contains(strings.ToLower(res.Detail), "deadline") {
			t.Fatalf("SSE result = %+v, want bounded deadline failure", res)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Doctor SSE check exceeded its behavior timeout")
	}
}

func TestInferenceCleanupCannotHangOnCancel(t *testing.T) {
	rt := &boundedDoctorRuntime{blockCancel: true}
	done := make(chan struct{}, 1)
	go func() {
		inferenceProbe(context.Background(), rt, 25*time.Millisecond)
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("inferenceProbe cleanup hung in Runtime.Cancel")
	}
}

func TestServicePublishesProgressBeforeRunningCheck(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	svc := NewService(checkFunc{name: "慢检查", category: CategoryConnection, fn: func(context.Context) (Status, string) {
		close(entered)
		<-release
		return Pass, "ok"
	}})

	started := make(chan string, 1)
	done := make(chan Report, 1)
	go func() {
		done <- svc.RunWithProgress(context.Background(), func(name string, _ Category) {
			started <- name
		}, nil)
	}()

	select {
	case name := <-started:
		if name != "慢检查" {
			t.Fatalf("progress name = %q", name)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Doctor did not publish progress before running the check")
	}
	select {
	case <-entered:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("check did not start")
	}
	close(release)
	<-done
}
