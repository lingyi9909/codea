package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodeaHomeDirDefaultAndOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
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
