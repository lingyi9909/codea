//go:build windows

package supervisor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

// processExists reports whether a process with the given pid is still alive,
// using OpenProcess with PROCESS_QUERY_INFORMATION.
func processExists(pid int) bool {
	h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = syscall.CloseHandle(h)
	return true
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

func TestStartRuntimeCommandRetriesTransientAccessDenied(t *testing.T) {
	attempts := 0
	var commands []*exec.Cmd

	cmd, err := startRuntimeCommandWith(
		func() *exec.Cmd {
			cmd := &exec.Cmd{}
			commands = append(commands, cmd)
			return cmd
		},
		func(*exec.Cmd) error {
			attempts++
			if attempts < 3 {
				return &os.PathError{Op: "fork/exec", Path: "opencode.exe", Err: syscall.ERROR_ACCESS_DENIED}
			}
			return nil
		},
		func(time.Duration) {},
	)
	if err != nil {
		t.Fatalf("startRuntimeCommandWith: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(commands) != 3 {
		t.Fatalf("commands = %d, want a fresh command per attempt", len(commands))
	}
	if commands[0] == commands[1] || commands[1] == commands[2] {
		t.Fatal("retry reused exec.Cmd; each Start attempt must use a fresh command")
	}
	if cmd != commands[2] {
		t.Fatal("returned command is not the successful attempt")
	}
}

func TestStartRuntimeCommandDoesNotRetryOtherErrors(t *testing.T) {
	attempts := 0
	wantErr := errors.New("bad executable format")

	cmd, err := startRuntimeCommandWith(
		func() *exec.Cmd { return &exec.Cmd{} },
		func(*exec.Cmd) error {
			attempts++
			return wantErr
		},
		func(time.Duration) {},
	)
	if cmd != nil {
		t.Fatal("command should be nil after failed start")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want exactly 1 for non-access-denied error", attempts)
	}
}

// TestStopTerminatesProcessTree is the Windows真机 gate for Blocking 1: the Job
// Object must guarantee Stop terminates opencode AND every descendant, leaving
// 0 orphans. Runs only on a real Windows machine.
func TestStopTerminatesProcessTree(t *testing.T) {
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
	s.mu.Lock()
	parentPID := s.cmd.Process.Pid
	s.mu.Unlock()

	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if s.Status() != runtime.RuntimeStopped {
		t.Fatalf("status = %s, want stopped", s.Status())
	}

	waitProcessGone(t, parentPID, 5*time.Second)
	waitProcessGone(t, childPID, 5*time.Second)
}

// TestStopForceKillFallback exercises the Job Object TerminateJobObject path
// when graceful CTRL_BREAK does not end the process within StopTimeout.
func TestStopForceKillFallback(t *testing.T) {
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
	waitProcessGone(t, pid, 5*time.Second)
}
