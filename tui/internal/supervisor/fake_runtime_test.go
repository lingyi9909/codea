package supervisor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

// fakeOpenCodeBin is the path to the compiled fake opencode server, built
// once in TestMain and reused across lifecycle tests.
var fakeOpenCodeBin string

func fakeExecutableName(base string) string {
	if goruntime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "codea-fake-opencode-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "mkdtemp: %v\n", err)
		os.Exit(1)
	}
	fakeOpenCodeBin = filepath.Join(tmp, fakeExecutableName("fake-opencode"))
	build := exec.Command("go", "build", "-o", fakeOpenCodeBin, "./fakeopencode")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build fake opencode: %v\n%s\n", err, out)
		os.RemoveAll(tmp)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// newTestSupervisor returns a Supervisor wired to the fake opencode binary.
// The fake runs in "healthy" mode by default; callers may add FAKE_OPENCODE_*
// env vars via t.Setenv before calling Start.
func newTestSupervisor(t *testing.T) *Supervisor {
	t.Helper()
	config := Config{
		OpenCodeBin:    fakeOpenCodeBin,
		Hostname:       "127.0.0.1",
		Port:           0,
		StartupTimeout: 15 * time.Second,
		StopTimeout:    5 * time.Second,
	}
	return NewSupervisor(config)
}

func waitForStatus(t *testing.T, s *Supervisor, want runtime.RuntimeStatus, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.Status() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("status = %s, want %s within %v", s.Status(), want, timeout)
}

// requireDarwin skips tests that exercise Unix process-group behaviour when
// not running on darwin (V1 supports macOS + Windows; Linux is deferred).
func requireDarwin(t *testing.T) {
	t.Helper()
	if goruntime.GOOS != "darwin" {
		t.Skipf("darwin-only process-group test, running on %s", goruntime.GOOS)
	}
}
