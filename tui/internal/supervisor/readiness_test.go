package supervisor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func TestProbeReadyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"healthy":true,"version":"1.18.11"}`))
	}))
	defer srv.Close()

	if err := probeReady(context.Background(), srv.URL, "opencode", "pw"); err != nil {
		t.Fatalf("probeReady: %v", err)
	}
}

func TestProbeReadySendsBasicAuth(t *testing.T) {
	var gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"healthy":true}`))
	}))
	defer srv.Close()

	if err := probeReady(context.Background(), srv.URL, "opencode", "s3cret"); err != nil {
		t.Fatalf("probeReady: %v", err)
	}
	if gotUser != "opencode" || gotPass != "s3cret" {
		t.Fatalf("BasicAuth = (%q,%q), want (opencode, s3cret)", gotUser, gotPass)
	}
}

func TestProbeReadyRejects401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if err := probeReady(context.Background(), srv.URL, "opencode", "wrong"); err == nil {
		t.Fatal("probeReady should not treat 401 as ready")
	}
}

func TestProbeReadyRejects500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if err := probeReady(context.Background(), srv.URL, "opencode", "pw"); err == nil {
		t.Fatal("probeReady should not treat 500 as ready")
	}
}

func TestProbeReadyRejectsUnhealthyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"healthy":false}`))
	}))
	defer srv.Close()

	if err := probeReady(context.Background(), srv.URL, "opencode", "pw"); err == nil {
		t.Fatal("probeReady should not treat healthy:false as ready")
	}
}

func TestFindFreePort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("port = %d, want valid port", port)
	}
}

func TestFixedPort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort: %v", err)
	}
	s := NewSupervisor(Config{
		OpenCodeBin:    fakeOpenCodeBin,
		Hostname:       "127.0.0.1",
		Port:           port,
		StartupTimeout: 15 * time.Second,
		StopTimeout:    5 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	if s.Port() != port {
		t.Fatalf("port = %d, want %d", s.Port(), port)
	}
}

func TestBaseURLAfterStart(t *testing.T) {
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer s.Stop()
	want := fmt.Sprintf("http://127.0.0.1:%d", s.Port())
	if got := s.BaseURL(); got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}
}

func TestStartWithAuthRequiredSucceeds(t *testing.T) {
	t.Setenv("FAKE_OPENCODE_REQUIRE_AUTH", "1")
	s := newTestSupervisor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start with auth-required fake should succeed: %v", err)
	}
	defer s.Stop()
	if s.Status() != runtime.RuntimeHealthy {
		t.Fatalf("status = %s, want healthy", s.Status())
	}
}

func TestStartupTimeoutCrashes(t *testing.T) {
	t.Setenv("FAKE_OPENCODE_MODE", "unhealthy")
	s := NewSupervisor(Config{
		OpenCodeBin:    fakeOpenCodeBin,
		Hostname:       "127.0.0.1",
		Port:           0,
		StartupTimeout: 500 * time.Millisecond,
		StopTimeout:    2 * time.Second,
	})
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("Start should fail when health never becomes ready")
	}
	if s.Status() != runtime.RuntimeCrashed {
		t.Fatalf("status = %s, want crashed", s.Status())
	}
}

func TestContextCancelDuringStart(t *testing.T) {
	t.Setenv("FAKE_OPENCODE_MODE", "never-ready")
	s := NewSupervisor(Config{
		OpenCodeBin:    fakeOpenCodeBin,
		Hostname:       "127.0.0.1",
		Port:           0,
		StartupTimeout: 30 * time.Second,
		StopTimeout:    2 * time.Second,
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Start(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Start should fail on context cancel")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}
}

func TestProcessExitsBeforeReadyFailsFast(t *testing.T) {
	t.Setenv("FAKE_OPENCODE_MODE", "exit-immediately")
	s := NewSupervisor(Config{
		OpenCodeBin:    fakeOpenCodeBin,
		Hostname:       "127.0.0.1",
		Port:           0,
		StartupTimeout: 30 * time.Second,
		StopTimeout:    2 * time.Second,
	})
	start := time.Now()
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("Start should fail when process exits immediately")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Start took %v, want fast failure on process exit", elapsed)
	}
	if s.Status() != runtime.RuntimeCrashed {
		t.Fatalf("status = %s, want crashed", s.Status())
	}
}
