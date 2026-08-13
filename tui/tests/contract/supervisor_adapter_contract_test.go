package contract

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
	"codea/tui/internal/supervisor"
)

// realOpenCodeBin resolves the real OpenCode binary used by the
// Supervisor↔Adapter integration contract. CODEA_OPENCODE_BIN overrides the
// default spike-artifacts location.
func realOpenCodeBin(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("CODEA_OPENCODE_BIN"); p != "" {
		return p
	}
	_, thisFile, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(repoRoot, "docs", "spike-artifacts", "opencode")
}

func requireRealOpenCode(t *testing.T) string {
	t.Helper()
	bin := realOpenCodeBin(t)
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("real OpenCode binary not available at %s: %v", bin, err)
	}
	return bin
}

// isolateOpenCodeEnv points HOME and XDG_* at a temp dir so the real OpenCode
// never reads/writes the developer's real config, cache or state — keeping the
// smoke offline and side-effect free.
func isolateOpenCodeEnv(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, ".cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, ".local", "state"))
}

// TestSupervisorAdapterContract proves the Supervisor-started Runtime can be
// driven by the OpenCodeAdapter using the exact BaseURL/Username/Password the
// Supervisor hands out — and that Stop tears the Runtime down and a restart
// issues fresh credentials that the Adapter picks up.
func TestSupervisorAdapterContract(t *testing.T) {
	bin := requireRealOpenCode(t)
	isolateOpenCodeEnv(t)

	s := supervisor.NewSupervisor(supervisor.Config{
		OpenCodeBin:    bin,
		Hostname:       "127.0.0.1",
		Port:           0,
		ConfigDir:      t.TempDir(),
		StartupTimeout: 60 * time.Second,
		StopTimeout:    10 * time.Second,
	})
	ctx := context.Background()

	// --- Start reaches Healthy ---
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s.Status() != runtime.RuntimeHealthy {
		t.Fatalf("status after Start = %s, want healthy", s.Status())
	}

	baseURL := s.BaseURL()
	username := s.Username()
	password := s.Password()
	if baseURL == "" || username == "" || password == "" {
		t.Fatalf("supervisor must expose BaseURL/Username/Password, got %q %q (pw len %d)",
			baseURL, username, len(password))
	}
	if username != "opencode" {
		t.Fatalf("username = %q, want opencode", username)
	}

	// --- Adapter drives the same Runtime with the same credentials ---
	rt := opencode.NewOpenCodeAdapter(baseURL, username, password)
	info, err := rt.Health(ctx)
	if err != nil {
		t.Fatalf("Health with supervisor credentials: %v", err)
	}
	if !info.Healthy {
		t.Fatal("Health: healthy=false")
	}

	// --- Wrong password is rejected ---
	bad := opencode.NewOpenCodeAdapter(baseURL, username, "wrong-password")
	if _, err := bad.Health(ctx); err == nil {
		t.Fatal("Health with wrong password should fail")
	} else if !runtime.IsAuth(err) {
		t.Fatalf("Health with wrong password: want auth error, got %v", err)
	}

	// --- Basic Runtime Contract: CreateSession + Subscribe ---
	session, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "supervisor contract"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("CreateSession: empty ID")
	}

	subCtx, subCancel := context.WithTimeout(ctx, 30*time.Second)
	defer subCancel()
	ch, err := rt.Subscribe(subCtx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if ch == nil {
		t.Fatal("Subscribe: nil channel")
	}
	seenConnected := false
	for evt := range ch {
		if evt.Type == runtime.EventType("runtime.connected") {
			seenConnected = true
			break
		}
	}
	if !seenConnected {
		t.Error("Subscribe: never received runtime.connected")
	}

	// --- Stop tears the Runtime down ---
	firstPassword := password
	if err := s.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if s.Status() != runtime.RuntimeStopped {
		t.Fatalf("status after Stop = %s, want stopped", s.Status())
	}
	if resp, err := http.Get(baseURL + "/global/health"); err == nil {
		resp.Body.Close()
		t.Fatalf("health still reachable after Stop at %s", baseURL)
	}

	// --- Restart issues fresh credentials; the Adapter picks them up ---
	if err := s.Start(ctx); err != nil {
		t.Fatalf("restart: %v", err)
	}
	defer s.Stop()
	if s.Status() != runtime.RuntimeHealthy {
		t.Fatalf("status after restart = %s, want healthy", s.Status())
	}
	newPassword := s.Password()
	if newPassword == "" || newPassword == firstPassword {
		t.Fatal("restart should generate a fresh password")
	}
	if p := s.Port(); p <= 0 {
		t.Fatalf("restart port = %d, want > 0", p)
	}

	rt2 := opencode.NewOpenCodeAdapter(s.BaseURL(), s.Username(), newPassword)
	if info2, err := rt2.Health(ctx); err != nil {
		t.Fatalf("Health after restart with new credentials: %v", err)
	} else if !info2.Healthy {
		t.Fatal("Health after restart: healthy=false")
	}

	// Stale pre-restart credentials must no longer work.
	stale := opencode.NewOpenCodeAdapter(s.BaseURL(), s.Username(), firstPassword)
	if _, err := stale.Health(ctx); err == nil {
		t.Fatal("Health with stale pre-restart password should fail")
	}
}
