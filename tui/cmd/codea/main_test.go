package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"codea/tui/internal/app"
	"codea/tui/internal/skill"
)

// fakeOpenCodeBin is the path to the compiled fake opencode server, built once
// in TestMain and reused across composition tests.
var fakeOpenCodeBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "codea-cmd-fake-opencode-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdtemp: %v\n", err)
		os.Exit(1)
	}
	fakeOpenCodeBin = filepath.Join(tmp, "fake-opencode")
	build := exec.Command("go", "build", "-o", fakeOpenCodeBin, "codea/tui/internal/supervisor/fakeopencode")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build fake opencode: %v\n%s\n", err, out)
		os.RemoveAll(tmp)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// TestBootstrapRuntimeSupervisedChain proves the product default chain:
// Supervisor.Start -> Healthy -> BaseURL/Username/Password -> Adapter -> App,
// then cleanup -> Stop. Requiring Basic Auth makes the credential wiring
// meaningful: the adapter's health check only succeeds if it presents the
// supervisor-generated username/password.
func TestBootstrapRuntimeSupervisedChain(t *testing.T) {
	t.Setenv("OPENCODE_BIN", fakeOpenCodeBin)
	t.Setenv("OPENCODE_URL", "") // force supervised path, not the dev override
	t.Setenv("FAKE_OPENCODE_REQUIRE_AUTH", "1")

	adapter, cleanup, err := bootstrapRuntime(t.TempDir())
	if err != nil {
		t.Fatalf("bootstrapRuntime: %v", err)
	}
	if adapter == nil {
		t.Fatal("adapter is nil on successful bootstrap")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := adapter.Health(ctx)
	if err != nil {
		t.Fatalf("adapter.Health: %v", err)
	}
	if !info.Healthy {
		t.Fatalf("runtime not healthy after supervised start: %+v", info)
	}

	if model := app.NewModel(adapter); model == nil {
		t.Fatal("app.NewModel returned nil")
	}

	cleanup()

	// After Stop the runtime is gone: health must fail, not report healthy.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if info2, err := adapter.Health(ctx2); err == nil && info2.Healthy {
		t.Fatal("runtime still healthy after cleanup (Stop did not terminate it)")
	}
}

// TestBootstrapRuntimeStartupFailure proves a Supervisor.Start error is
// surfaced (明确报错) and does not hand the caller a fake "ready" adapter.
func TestBootstrapRuntimeStartupFailure(t *testing.T) {
	t.Setenv("OPENCODE_BIN", fakeOpenCodeBin)
	t.Setenv("OPENCODE_URL", "")
	t.Setenv("FAKE_OPENCODE_MODE", "exit-immediately")

	adapter, cleanup, err := bootstrapRuntime(t.TempDir())
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatal("bootstrapRuntime should fail when the runtime exits immediately")
	}
	if adapter != nil {
		t.Error("adapter must be nil on startup failure")
	}
	if err.Error() == "" {
		t.Error("startup failure error must be non-empty")
	}
}

// TestCodeaConfigDirDefaultsIsolated guards P0-1: the default controlled config
// dir must be a dedicated Codea location, never OpenCode's native ~/.config/opencode.
func TestCodeaConfigDirDefaultsIsolated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEA_RUNTIME_CONFIG_DIR", "")

	got := codeaConfigDir()
	want := filepath.Join(home, ".codea", "runtime-config")
	if got != want {
		t.Fatalf("codeaConfigDir = %q, want %q", got, want)
	}
	if got == filepath.Join(home, ".config", "opencode") {
		t.Fatal("codeaConfigDir must not be OpenCode's native ~/.config/opencode")
	}
}

func TestCodeaConfigDirHonorsOverride(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "custom")
	t.Setenv("CODEA_RUNTIME_CONFIG_DIR", custom)
	if got := codeaConfigDir(); got != custom {
		t.Fatalf("codeaConfigDir = %q, want %q", got, custom)
	}
}

// TestSkillRootsTreatsUserOpenCodeAsReadOnly guards P0-1: the user's native
// OpenCode skills dir must appear as a read-only SourceUser root, never as the
// Codea sync target.
func TestSkillRootsTreatsUserOpenCodeAsReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEA_SKILLS_DIR", filepath.Join(t.TempDir(), "dist"))

	roots := skillRoots()

	found := false
	for _, r := range roots {
		if r.Source == skill.SourceUser && r.Dir == filepath.Join(home, ".config", "opencode", "skills") {
			found = true
		}
	}
	if !found {
		t.Fatalf("~/.config/opencode/skills missing as SourceUser root: %+v", roots)
	}
}
