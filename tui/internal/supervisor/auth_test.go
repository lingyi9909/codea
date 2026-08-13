package supervisor

import (
	"strings"
	"testing"
)

func envValue(env []string, key string) (string, bool) {
	prefix := key + "="
	val := ""
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			val = strings.TrimPrefix(e, prefix)
			found = true
		}
	}
	return val, found
}

func TestGeneratePasswordNonEmpty(t *testing.T) {
	pw, err := generatePassword()
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	if pw == "" {
		t.Fatal("password must not be empty")
	}
}

func TestGeneratePasswordIs32RandomBytes(t *testing.T) {
	pw, err := generatePassword()
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	// 32 bytes -> 64 hex chars.
	if len(pw) != 64 {
		t.Fatalf("password length = %d, want 64", len(pw))
	}
	for _, r := range pw {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("password contains non-hex rune %q", r)
		}
	}
}

func TestGeneratePasswordUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		pw, err := generatePassword()
		if err != nil {
			t.Fatalf("generatePassword: %v", err)
		}
		if seen[pw] {
			t.Fatalf("duplicate password generated: %q", pw)
		}
		seen[pw] = true
	}
}

func TestUsernameDefaultsToOpencode(t *testing.T) {
	s := newTestSupervisor(t)
	if got := s.Username(); got != "opencode" {
		t.Fatalf("Username() = %q, want %q", got, "opencode")
	}
}

func TestHostnameDefaultsToLocalhost(t *testing.T) {
	s := NewSupervisor(Config{})
	if s.config.Hostname != "127.0.0.1" {
		t.Fatalf("default hostname = %q, want 127.0.0.1", s.config.Hostname)
	}
}

func TestBuildEnvCarriesCredentials(t *testing.T) {
	cfg := Config{ConfigDir: "/tmp/codea-config"}
	env := buildEnv(cfg, "opencode", "s3cret-password")

	if v, ok := envValue(env, "OPENCODE_SERVER_USERNAME"); !ok || v != "opencode" {
		t.Fatalf("OPENCODE_SERVER_USERNAME = %q (ok=%v), want opencode", v, ok)
	}
	if v, ok := envValue(env, "OPENCODE_SERVER_PASSWORD"); !ok || v != "s3cret-password" {
		t.Fatalf("OPENCODE_SERVER_PASSWORD = %q (ok=%v), want s3cret-password", v, ok)
	}
}

func TestBuildEnvCarriesConfigDir(t *testing.T) {
	cfg := Config{ConfigDir: "/tmp/codea-config"}
	env := buildEnv(cfg, "opencode", "pw")
	if v, ok := envValue(env, "OPENCODE_CONFIG_DIR"); !ok || v != "/tmp/codea-config" {
		t.Fatalf("OPENCODE_CONFIG_DIR = %q (ok=%v), want /tmp/codea-config", v, ok)
	}
}

func TestBuildEnvCarriesOfflineVars(t *testing.T) {
	env := buildEnv(Config{}, "opencode", "pw")
	offline := []string{
		"OPENCODE_DISABLE_CLAUDE_CODE",
		"OPENCODE_DISABLE_MODELS_FETCH",
		"OPENCODE_DISABLE_AUTOUPDATE",
		"OPENCODE_DISABLE_EMBEDDED_WEB_UI",
		"OPENCODE_DISABLE_LSP_DOWNLOAD",
		"OPENCODE_DISABLE_DEFAULT_PLUGINS",
	}
	for _, key := range offline {
		if v, ok := envValue(env, key); !ok || v != "1" {
			t.Fatalf("%s = %q (ok=%v), want 1", key, v, ok)
		}
	}
}

func TestBuildArgsNeverContainsPassword(t *testing.T) {
	const password = "s3cret-password"
	args := buildArgs(Config{Hostname: "127.0.0.1"}, 12345)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, password) {
		t.Fatalf("args must not contain the password: %v", args)
	}
}

func TestBuildArgsShape(t *testing.T) {
	args := buildArgs(Config{Hostname: "127.0.0.1"}, 12345)
	want := []string{"serve", "--hostname", "127.0.0.1", "--port", "12345"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestBuildArgsCannotExposeRuntime(t *testing.T) {
	args := buildArgs(Config{Hostname: "0.0.0.0"}, 12345)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "0.0.0.0") {
		t.Fatalf("buildArgs exposed a wildcard bind: %v", args)
	}
	if !strings.Contains(joined, "127.0.0.1") {
		t.Fatalf("buildArgs must hard-lock loopback: %v", args)
	}
}

func TestSupervisorForcesLoopback(t *testing.T) {
	s := NewSupervisor(Config{Hostname: "0.0.0.0"})
	if s.config.Hostname != "127.0.0.1" {
		t.Fatalf("hostname = %q, want forced loopback 127.0.0.1", s.config.Hostname)
	}
	args := buildArgs(s.config, 12345)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "0.0.0.0") {
		t.Fatalf("supervisor would bind wildcard: %v", args)
	}
}
