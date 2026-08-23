package update

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUpgradePreCheckFailureCleansTransactionScratch(t *testing.T) {
	home := makeHomeWithV1(t)
	release := buildRelease(t, filepath.Join(t.TempDir(), "v2"), "2.0.0", 1)
	s := newService(t, home, &fakeChecker{failPhase: CheckPreSwitch}, NewMigrationRegistry())
	if err := s.Upgrade(context.Background(), release); err == nil {
		t.Fatal("expected pre-switch failure")
	}
	assertDirEmptyOrMissing(t, filepath.Join(home, "transactions"))
	assertDirEmptyOrMissing(t, filepath.Join(home, "staging"))
}

func TestSuccessfulUpgradeCleansTransactionScratch(t *testing.T) {
	home := makeHomeWithV1(t)
	release := buildRelease(t, filepath.Join(t.TempDir(), "v2"), "2.0.0", 1)
	s := newService(t, home, &fakeChecker{}, NewMigrationRegistry())
	if err := s.Upgrade(context.Background(), release); err != nil {
		t.Fatal(err)
	}
	assertDirEmptyOrMissing(t, filepath.Join(home, "transactions"))
	assertDirEmptyOrMissing(t, filepath.Join(home, "staging"))
}

func assertDirEmptyOrMissing(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s contains stale transaction data: %v", dir, entries)
	}
}
