//go:build windows

package supervisor

import (
	"fmt"
	"os/exec"
	"syscall"
)

// configureProcess places opencode in a new process group so the whole tree
// can be signalled together.
func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// terminateProcess sends CTRL_BREAK_EVENT to the process group as a graceful
// shutdown request. It is best-effort: a console-less process may not receive
// it, in which case the caller falls back to killProcess after StopTimeout.
func terminateProcess(cmd *exec.Cmd) error {
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	defer dll.Release()
	proc, err := dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return err
	}
	r1, _, lastErr := proc.Call(syscall.CTRL_BREAK_EVENT, uintptr(cmd.Process.Pid))
	if r1 == 0 {
		if lastErr != nil {
			return lastErr
		}
		return fmt.Errorf("GenerateConsoleCtrlEvent failed")
	}
	return nil
}

// killProcess force-terminates the process (TerminateProcess).
func killProcess(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
}
