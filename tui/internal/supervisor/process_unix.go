//go:build !windows

package supervisor

import (
	"os/exec"
	"syscall"
)

// configureProcess places opencode in its own process group so the whole
// process tree (opencode + children) can be signalled as one unit.
func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// attachProcess is a no-op on Unix: process-group signalling needs no extra
// OS handle to be tracked.
func attachProcess(cmd *exec.Cmd) error { return nil }

// detachProcess is a no-op on Unix.
func detachProcess(cmd *exec.Cmd) {}

// terminateProcess sends SIGTERM to the process group (graceful shutdown).
func terminateProcess(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

// killProcess sends SIGKILL to the process group (force kill fallback).
func killProcess(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
