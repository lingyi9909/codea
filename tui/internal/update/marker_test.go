package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateLockPublishesLaunchBlocker(t *testing.T) {
	home := t.TempDir()
	marker := filepath.Join(home, "update.in-progress")
	lock, err := acquireUpdateLock(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		_ = lock.Release()
		t.Fatalf("update marker not published while lock is held: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("update marker remains after release: %v", err)
	}
}

func TestUpdateLockReplacesStaleLaunchBlocker(t *testing.T) {
	home := t.TempDir()
	marker := filepath.Join(home, "update.in-progress")
	if err := os.WriteFile(marker, []byte("stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireUpdateLock(home)
	if err != nil {
		t.Fatalf("stale marker must not prevent crash recovery: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("stale marker not cleaned after recovered transaction: %v", err)
	}
}
