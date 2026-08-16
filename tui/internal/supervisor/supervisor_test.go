package supervisor

import (
	"context"
	"sync"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func TestDefaultStatusStopped(t *testing.T) {
	s := newTestSupervisor(t)
	if got := s.Status(); got != runtime.RuntimeStopped {
		t.Fatalf("initial status = %s, want %s", got, runtime.RuntimeStopped)
	}
	if s.Port() != 0 {
		t.Fatalf("initial port = %d, want 0", s.Port())
	}
}

func TestStartReachesHealthy(t *testing.T) {
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()

	if got := s.Status(); got != runtime.RuntimeHealthy {
		t.Fatalf("status after Start = %s, want %s", got, runtime.RuntimeHealthy)
	}
	if s.Port() <= 0 {
		t.Fatalf("port = %d, want > 0", s.Port())
	}
	if got := s.BaseURL(); got == "" {
		t.Fatal("BaseURL must not be empty")
	}
}

func TestStartWhileHealthyErrors(t *testing.T) {
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	defer s.Stop()

	if err := s.Start(ctx); err == nil {
		t.Fatal("second Start while healthy should error")
	}
	// State must remain Healthy, not be corrupted by the rejected Start.
	if got := s.Status(); got != runtime.RuntimeHealthy {
		t.Fatalf("status after rejected Start = %s, want %s", got, runtime.RuntimeHealthy)
	}
}

func TestStopIdempotent(t *testing.T) {
	s := newTestSupervisor(t)

	// Stop before any Start is a no-op.
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop before Start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if got := s.Status(); got != runtime.RuntimeStopped {
		t.Fatalf("status after Stop = %s, want %s", got, runtime.RuntimeStopped)
	}

	// Second Stop is idempotent and must not panic.
	if err := s.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if got := s.Status(); got != runtime.RuntimeStopped {
		t.Fatalf("status after second Stop = %s, want %s", got, runtime.RuntimeStopped)
	}
}

func TestUnexpectedExitCrashes(t *testing.T) {
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s.Status() != runtime.RuntimeHealthy {
		t.Fatalf("want healthy before crash, got %s", s.Status())
	}

	// Simulate the OpenCode process being killed externally.
	s.mu.Lock()
	proc := s.cmd.Process
	s.mu.Unlock()
	if err := proc.Kill(); err != nil {
		t.Fatalf("external kill: %v", err)
	}

	waitForStatus(t, s, runtime.RuntimeCrashed, 5*time.Second)
	if s.LastError() == nil {
		t.Fatal("expected a non-nil LastError after unexpected exit")
	}
}

func TestConcurrentStartSingleProcess(t *testing.T) {
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	const n = 8
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = s.Start(ctx)
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 successful Start, got %d (errs=%v)", successes, errs)
	}
	defer s.Stop()

	if got := s.Status(); got != runtime.RuntimeHealthy {
		t.Fatalf("status = %s, want %s", got, runtime.RuntimeHealthy)
	}
}

func TestHealthyThenExitSettlesCrashed(t *testing.T) {
	t.Setenv("FAKE_OPENCODE_MODE", "healthy-then-exit")
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Start may return nil (raced to Healthy before the process died) or an
	// error (process exited before the CAS check). Either way the process is
	// gone, and the supervisor must reconcile to Crashed — never a stale Healthy.
	_ = s.Start(ctx)

	waitForStatus(t, s, runtime.RuntimeCrashed, 5*time.Second)
}

func TestRestartAfterStop(t *testing.T) {
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	firstPassword := s.Password()
	firstPort := s.Port()
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if err := s.Start(ctx); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	defer s.Stop()

	if got := s.Status(); got != runtime.RuntimeHealthy {
		t.Fatalf("status after restart = %s, want %s", got, runtime.RuntimeHealthy)
	}
	if s.Password() == firstPassword {
		t.Fatal("restart must generate a new password")
	}
	if s.Port() == 0 {
		t.Fatal("restart must have a valid port")
	}
	// Port may or may not differ (auto-allocation), but must be valid either way.
	_ = firstPort
}

func TestMarkHealthyAcceptsCurrentStarting(t *testing.T) {
	s := newTestSupervisor(t)
	s.mu.Lock()
	s.status = runtime.RuntimeStarting
	s.runID = 7
	s.mu.Unlock()

	if !s.markHealthy(7) {
		t.Fatal("markHealthy should accept the current run in Starting state")
	}
	if got := s.Status(); got != runtime.RuntimeHealthy {
		t.Fatalf("status = %s, want %s", got, runtime.RuntimeHealthy)
	}
}

func TestMarkHealthyRejectsAfterCrash(t *testing.T) {
	s := newTestSupervisor(t)
	s.mu.Lock()
	s.status = runtime.RuntimeCrashed
	s.runID = 7
	s.mu.Unlock()

	if s.markHealthy(7) {
		t.Fatal("markHealthy must not overwrite Crashed with Healthy")
	}
	if got := s.Status(); got != runtime.RuntimeCrashed {
		t.Fatalf("status = %s, want %s", got, runtime.RuntimeCrashed)
	}
}

func TestMarkHealthyRejectsStaleRun(t *testing.T) {
	s := newTestSupervisor(t)
	s.mu.Lock()
	s.status = runtime.RuntimeStarting
	s.runID = 9 // current run
	s.mu.Unlock()

	if s.markHealthy(7) {
		t.Fatal("markHealthy must reject a stale runID")
	}
	if got := s.Status(); got != runtime.RuntimeStarting {
		t.Fatalf("status = %s, want %s", got, runtime.RuntimeStarting)
	}
}

func hasEnv(env []string, k string) bool {
	for _, e := range env {
		if e == k {
			return true
		}
	}
	return false
}

func TestBuildEnvIsolation(t *testing.T) {
	base := buildEnv(Config{ConfigDir: "/c", CodeaSkillsOnly: false}, "u", "p")
	if hasEnv(base, "OPENCODE_DISABLE_EXTERNAL_SKILLS=1") || hasEnv(base, "OPENCODE_DISABLE_PROJECT_CONFIG=1") {
		t.Fatal("compatible mode must not disable external/project skills")
	}
	if !hasEnv(base, "OPENCODE_DISABLE_CLAUDE_CODE=1") {
		t.Fatal("Task 1 offline lock must remain")
	}

	strict := buildEnv(Config{ConfigDir: "/c", CodeaSkillsOnly: true}, "u", "p")
	if !hasEnv(strict, "OPENCODE_DISABLE_EXTERNAL_SKILLS=1") || !hasEnv(strict, "OPENCODE_DISABLE_PROJECT_CONFIG=1") {
		t.Fatal("strict mode must disable external + project skills")
	}
}
