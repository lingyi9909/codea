//go:build darwin

package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func processExists(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

func waitProcessGone(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("process %d still alive after %v", pid, timeout)
}

func TestStopGraceful(t *testing.T) {
	requireDarwin(t)
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	s.mu.Lock()
	pid := s.cmd.Process.Pid
	s.mu.Unlock()

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if s.Status() != runtime.RuntimeStopped {
		t.Fatalf("status = %s, want stopped", s.Status())
	}
	waitProcessGone(t, pid, 5*time.Second)
}

func TestStopForceKillFallback(t *testing.T) {
	requireDarwin(t)
	t.Setenv("FAKE_OPENCODE_IGNORE_SIGTERM", "1")
	s := NewSupervisor(Config{
		OpenCodeBin:    fakeOpenCodeBin,
		Hostname:       "127.0.0.1",
		Port:           0,
		StartupTimeout: 15 * time.Second,
		StopTimeout:    500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	s.mu.Lock()
	pid := s.cmd.Process.Pid
	s.mu.Unlock()

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if s.Status() != runtime.RuntimeStopped {
		t.Fatalf("status = %s, want stopped", s.Status())
	}
	// The process ignored SIGTERM, so only the force-kill fallback could end it.
	waitProcessGone(t, pid, 5*time.Second)
}

func TestChildProcessNotLeftBehind(t *testing.T) {
	requireDarwin(t)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	t.Setenv("FAKE_OPENCODE_SPAWN_CHILD", "1")
	t.Setenv("FAKE_OPENCODE_CHILD_PID_FILE", pidFile)

	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	childPID := waitChildPID(t, pidFile, 5*time.Second)
	if !processExists(childPID) {
		t.Fatalf("child %d should be alive before Stop", childPID)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitProcessGone(t, childPID, 5*time.Second)
}

func TestStopAfterCrashNoPanic(t *testing.T) {
	requireDarwin(t)
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	s.mu.Lock()
	proc := s.cmd.Process
	s.mu.Unlock()
	_ = proc.Kill()
	waitForStatus(t, s, runtime.RuntimeCrashed, 5*time.Second)

	// Stop on an already-crashed supervisor must be a safe no-op.
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop after crash: %v", err)
	}
	if s.Status() != runtime.RuntimeCrashed {
		t.Fatalf("status = %s, want crashed to be preserved", s.Status())
	}
}

func waitChildPID(t *testing.T, pidFile string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidFile)
		if err == nil {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if convErr == nil && pid > 0 {
				return pid
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("child PID file %s never appeared", pidFile)
	return 0
}
