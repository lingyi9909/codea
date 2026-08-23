//go:build !windows

package update

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type platformSwitcher struct{ home string }

func newPlatformSwitcher(home string) Switcher { return &platformSwitcher{home: home} }
func (s *platformSwitcher) Current() (string, error) {
	p := filepath.Join(s.home, "current")
	target, err := os.Readlink(p)
	if err != nil { return "", err }
	if !filepath.IsAbs(target) { target = filepath.Join(filepath.Dir(p), target) }
	abs, err := validateVersionTarget(s.home, target)
	if err != nil { return "", fmt.Errorf("invalid current pointer: %w", err) }
	return abs, nil
}
func (s *platformSwitcher) Switch(target string) error {
	abs, err := validateVersionTarget(s.home, target)
	if err != nil { return err }
	if err := os.MkdirAll(s.home, 0o755); err != nil { return err }
	current := filepath.Join(s.home, "current")
	tmp := current + ".next"
	_ = os.Remove(tmp)
	if err := os.Symlink(abs, tmp); err != nil { return err }
	if err := os.Rename(tmp, current); err != nil { _ = os.Remove(tmp); return fmt.Errorf("switch current: %w", err) }
	return nil
}

func replaceFileAtomic(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

type fileLock struct {
	f      *os.File
	marker string
}

func acquireUpdateLock(home string) (updateLock, error) {
	if err := os.MkdirAll(home, 0o700); err != nil { return nil, err }
	f, err := os.OpenFile(filepath.Join(home, "update.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil { return nil, err }
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil { f.Close(); return nil, fmt.Errorf("another update is running: %w", err) }
	marker := filepath.Join(home, "update.in-progress")
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("pid=%d\n", os.Getpid())), 0o600); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, fmt.Errorf("publish update marker: %w", err)
	}
	return &fileLock{f: f, marker: marker}, nil
}
func (l *fileLock) Release() error {
	if l == nil || l.f == nil { return nil }
	markerErr := os.Remove(l.marker)
	if os.IsNotExist(markerErr) { markerErr = nil }
	err1 := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	err2 := l.f.Close()
	l.f = nil
	if markerErr != nil { return markerErr }
	if err1 != nil { return err1 }
	return err2
}
func (l *fileLock) Close() error { return l.Release() }
