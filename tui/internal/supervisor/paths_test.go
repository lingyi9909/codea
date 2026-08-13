package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func TestBuildEnvConfigDirWithSpaces(t *testing.T) {
	const dir = "/tmp/codea config dir with spaces"
	env := buildEnv(Config{ConfigDir: dir}, "opencode", "pw")
	if v, ok := envValue(env, "OPENCODE_CONFIG_DIR"); !ok || v != dir {
		t.Fatalf("OPENCODE_CONFIG_DIR = %q (ok=%v), want %q verbatim", v, ok, dir)
	}
}

func TestStartWithSpacesInPaths(t *testing.T) {
	base := t.TempDir()
	binDir := filepath.Join(base, "bin dir with spaces")
	projectRoot := filepath.Join(base, "project dir with spaces")
	configDir := filepath.Join(base, "config dir with spaces")
	for _, d := range []string{binDir, projectRoot, configDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	bin := filepath.Join(binDir, "fake opencode with spaces")
	data, err := os.ReadFile(fakeOpenCodeBin)
	if err != nil {
		t.Fatalf("read fake binary: %v", err)
	}
	if err := os.WriteFile(bin, data, 0o755); err != nil {
		t.Fatalf("copy fake binary: %v", err)
	}

	s := NewSupervisor(Config{
		OpenCodeBin:    bin,
		Hostname:       "127.0.0.1",
		Port:           0,
		ConfigDir:      configDir,
		ProjectRoot:    projectRoot,
		StartupTimeout: 15 * time.Second,
		StopTimeout:    5 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start with spaces in bin/ConfigDir/ProjectRoot: %v", err)
	}
	defer s.Stop()
	if got := s.Status(); got != runtime.RuntimeHealthy {
		t.Fatalf("status = %s, want %s", got, runtime.RuntimeHealthy)
	}
}
