//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codea/tui/internal/skill"
)

func TestRealOpenCodePluginAgentWorksWithOfflineNPM(t *testing.T) {
	bin := os.Getenv("CODEA_REAL_OPENCODE_BIN")
	bundle := os.Getenv("CODEA_REAL_PLUGIN_BUNDLE")
	if bin == "" || bundle == "" {
		t.Skip("real OpenCode integration inputs not configured")
	}

	cfgDir := t.TempDir()
	projectRoot := t.TempDir()
	// Exercise the remaining OpenCode dependency-install path too: compatible
	// mode deliberately keeps project .opencode discovery enabled. This directory
	// has no sentinel, so the bounded npm settings must make its unreachable
	// registry attempt fail quickly instead of blocking /agent.
	if err := os.MkdirAll(filepath.Join(projectRoot, ".opencode"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENCODE_BIN", bin)
	t.Setenv("OPENCODE_URL", "")
	t.Setenv("CODEA_PLUGIN_BUNDLE", bundle)
	// TEST-NET-1 is deliberately non-routable on the public Internet. If the
	// Codea-owned dependency sentinels regress, or project dependency retries are
	// left unbounded, this test reproduces the production /agent hang.
	t.Setenv("npm_config_registry", "http://192.0.2.1:81")

	if err := writePluginConfig(cfgDir); err != nil {
		t.Fatalf("writePluginConfig: %v", err)
	}
	for _, dir := range []string{
		cfgDir,
		filepath.Join(cfgDir, "xdg", "config", "opencode"),
	} {
		if _, err := os.Stat(filepath.Join(dir, "package-lock.json")); err != nil {
			t.Fatalf("offline dependency sentinel missing in %s: %v", dir, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adapter, cleanup, err := bootstrapRuntimeAt(cfgDir, skill.SkillModeCompatible, projectRoot)
	if err != nil {
		t.Fatalf("bootstrapRuntimeAt: %v", err)
	}
	defer cleanup()

	started := time.Now()
	agents, err := adapter.ListAgents(ctx)
	elapsed := time.Since(started)
	if err != nil {
		t.Fatalf("ListAgents after %v: %v", elapsed, err)
	}
	if len(agents) == 0 {
		t.Fatal("ListAgents returned no agents")
	}
	if elapsed > 20*time.Second {
		t.Fatalf("ListAgents took %v with offline npm; want <=20s", elapsed)
	}
	t.Logf("REAL_OFFLINE_PLUGIN_AGENT PASS agents=%d elapsed=%v", len(agents), elapsed)
}
