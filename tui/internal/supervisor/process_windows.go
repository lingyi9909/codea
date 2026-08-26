//go:build windows

package supervisor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
	procGenerateConsoleCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

const (
	jobObjectExtendedLimitInformationClass = 9
	jobObjectLimitKillOnJobClose           = 0x00002000
	processAllAccess                       = 0x001F0FFF
)

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// jobHandles maps each started *exec.Cmd to its Job Object handle so
// detachProcess/killProcess can reach it after start.
var jobHandles sync.Map // *exec.Cmd -> syscall.Handle

// prepareRuntimeBinary removes the Mark-of-the-Web alternate data stream from
// the verified bundled runtime immediately before execution. The installer also
// unblocks verified files, but some Windows environments can preserve or
// re-apply Zone.Identifier after extraction/copy. Runtime launch therefore
// enforces the invariant again at the final execution boundary.
func prepareRuntimeBinary(path string) error {
	zoneIdentifier := path + ":Zone.Identifier"
	if err := os.Remove(zoneIdentifier); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove Zone.Identifier from %s: %w", path, err)
	}
	return nil
}

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// attachProcess places the started process into a fresh Job Object configured
// with KILL_ON_JOB_CLOSE. Closing that handle (detachProcess) then terminates
// every remaining descendant, so a crashed opencode can never orphan children.
func attachProcess(cmd *exec.Cmd) error {
	job, _, lastErr := procCreateJobObjectW.Call(0, 0)
	if job == 0 {
		return win32Err("CreateJobObjectW", lastErr)
	}

	var info jobObjectExtendedLimitInformation
	info.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	r1, _, lastErr := procSetInformationJobObject.Call(
		job,
		jobObjectExtendedLimitInformationClass,
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	if r1 == 0 {
		_ = closeHandle(job)
		return win32Err("SetInformationJobObject", lastErr)
	}

	process, _, lastErr := procOpenProcess.Call(processAllAccess, 0, uintptr(cmd.Process.Pid))
	if process == 0 {
		_ = closeHandle(job)
		return win32Err("OpenProcess", lastErr)
	}

	r1, _, lastErr = procAssignProcessToJobObject.Call(job, process)
	_ = closeHandle(process)
	if r1 == 0 {
		_ = closeHandle(job)
		return win32Err("AssignProcessToJobObject", lastErr)
	}

	jobHandles.Store(cmd, syscall.Handle(job))
	return nil
}

// detachProcess closes the Job Object handle. With KILL_ON_JOB_CLOSE set, this
// force-terminates any remaining process in the job (parent + descendants).
func detachProcess(cmd *exec.Cmd) {
	if v, ok := jobHandles.LoadAndDelete(cmd); ok {
		_ = closeHandle(uintptr(v.(syscall.Handle)))
	}
}

// terminateProcess sends CTRL_BREAK_EVENT to the process group as a graceful
// shutdown request. Best-effort: a console-less process may ignore it, in which
// case the caller falls back to killProcess after StopTimeout.
func terminateProcess(cmd *exec.Cmd) error {
	r1, _, lastErr := procGenerateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(cmd.Process.Pid))
	if r1 == 0 {
		return win32Err("GenerateConsoleCtrlEvent", lastErr)
	}
	return nil
}

// killProcess terminates the entire Job Object (parent + descendants) and
// closes the handle, guaranteeing no process from this run survives.
func killProcess(cmd *exec.Cmd) {
	if v, ok := jobHandles.LoadAndDelete(cmd); ok {
		handle := v.(syscall.Handle)
		_, _, _ = procTerminateJobObject.Call(uintptr(handle), 1)
		_ = closeHandle(uintptr(handle))
	}
}

func closeHandle(h uintptr) error {
	r1, _, lastErr := procCloseHandle.Call(h)
	if r1 == 0 {
		return win32Err("CloseHandle", lastErr)
	}
	return nil
}

func win32Err(api string, lastErr error) error {
	if lastErr == nil {
		return fmt.Errorf("%s failed", api)
	}
	if errno, ok := lastErr.(syscall.Errno); ok && errno == 0 {
		return fmt.Errorf("%s failed", api)
	}
	return fmt.Errorf("%s: %w", api, lastErr)
}
