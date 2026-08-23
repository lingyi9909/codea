//go:build windows

package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type platformSwitcher struct{ home string }

func newPlatformSwitcher(home string) Switcher { return &platformSwitcher{home: home} }
func (s *platformSwitcher) Current() (string, error) {
	b, err := os.ReadFile(filepath.Join(s.home, "current.txt"))
	if err != nil { return "", err }
	target := strings.TrimSpace(string(b))
	if target == "" { return "", fmt.Errorf("current.txt is empty") }
	abs, err := validateVersionTarget(s.home, target)
	if err != nil { return "", fmt.Errorf("invalid current pointer: %w", err) }
	return abs, nil
}
func (s *platformSwitcher) Switch(target string) error {
	abs, err := validateVersionTarget(s.home, target)
	if err != nil { return err }
	return writeFileAtomic(filepath.Join(s.home, "current.txt"), []byte(abs+"\r\n"), 0o600)
}

const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
	movefileReplaceExisting = 0x1
	movefileWriteThrough    = 0x8
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
	procMoveFileExW  = kernel32.NewProc("MoveFileExW")
)

type fileLock struct {
	f      *os.File
	ov     syscall.Overlapped
	marker string
}

func acquireUpdateLock(home string) (updateLock, error) {
	if err := os.MkdirAll(home, 0o700); err != nil { return nil, err }
	f, err := os.OpenFile(filepath.Join(home, "update.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil { return nil, err }
	l := &fileLock{f: f}
	r, _, callErr := procLockFileEx.Call(f.Fd(), lockfileFailImmediately|lockfileExclusiveLock, 0, 1, 0, uintptr(unsafe.Pointer(&l.ov)))
	if r == 0 { f.Close(); return nil, fmt.Errorf("another update is running: %w", callErr) }
	l.marker = filepath.Join(home, "update.in-progress")
	if err := os.WriteFile(l.marker, []byte(fmt.Sprintf("pid=%d\r\n", os.Getpid())), 0o600); err != nil {
		_, _, _ = procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&l.ov)))
		_ = f.Close()
		return nil, fmt.Errorf("publish update marker: %w", err)
	}
	return l, nil
}
func (l *fileLock) Release() error {
	if l == nil || l.f == nil { return nil }
	markerErr := os.Remove(l.marker)
	if os.IsNotExist(markerErr) { markerErr = nil }
	r, _, callErr := procUnlockFileEx.Call(l.f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&l.ov)))
	closeErr := l.f.Close()
	l.f = nil
	if markerErr != nil { return markerErr }
	if r == 0 { return callErr }
	return closeErr
}
func (l *fileLock) Close() error { return l.Release() }
func replaceFileAtomic(oldPath, newPath string) error {
	oldp, err := syscall.UTF16PtrFromString(oldPath); if err != nil { return err }
	newp, err := syscall.UTF16PtrFromString(newPath); if err != nil { return err }
	r, _, callErr := procMoveFileExW.Call(uintptr(unsafe.Pointer(oldp)), uintptr(unsafe.Pointer(newp)), movefileReplaceExisting|movefileWriteThrough)
	if r == 0 { return callErr }
	return nil
}
