package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCodeaHomeDirDefaultAndOverride(t *testing.T) {
	home := t.TempDir()
	setUserHomeForTest(t, home)
	t.Setenv("CODEA_HOME", "")
	if got := codeaHomeDir(); got != filepath.Join(home, ".codea") { t.Fatalf("default=%s", got) }
	custom := filepath.Join(t.TempDir(), "custom")
	t.Setenv("CODEA_HOME", custom)
	if got := codeaHomeDir(); got != custom { t.Fatalf("override=%s", got) }
}

func TestRunInitCommandCreatesConfig(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "cfg")
	t.Setenv("CODEA_HOME", home)
	t.Setenv("CODEA_RUNTIME_CONFIG_DIR", cfg)
	if err := runInitCommand(); err != nil { t.Fatal(err) }
	if _, err := os.Stat(filepath.Join(cfg, "codea", "config.json")); err != nil { t.Fatal(err) }
}

func TestRecoverInterruptedUpdateIfNeededClearsStaleMarkerWithoutJournal(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, "runtime-config")
	t.Setenv("CODEA_HOME", home)
	t.Setenv("CODEA_RUNTIME_CONFIG_DIR", cfg)
	marker := filepath.Join(home, "update.in-progress")
	if err := os.WriteFile(marker, []byte("stale\n"), 0o600); err != nil { t.Fatal(err) }
	if err := recoverInterruptedUpdateIfNeeded(context.Background()); err != nil { t.Fatal(err) }
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("stale marker remains after recovery: %v", err)
	}
}
