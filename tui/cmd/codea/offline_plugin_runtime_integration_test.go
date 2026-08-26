package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"codea/tui/internal/skill"
)

func TestRealOpenCodePluginAgentLoadDoesNotTouchNpmRegistry(t *testing.T) {
	openCodeBin := os.Getenv("CODEA_REAL_OPENCODE_BIN")
	bundle := os.Getenv("CODEA_REAL_PLUGIN_BUNDLE")
	if openCodeBin == "" || bundle == "" {
		t.Skip("real OpenCode/plugin paths not provided")
	}
	if _, err := os.Stat(openCodeBin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bundle); err != nil {
		t.Fatal(err)
	}

	var registryHits atomic.Int32
	var registryMu sync.Mutex
	var registryRequests []string
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registryHits.Add(1)
		registryMu.Lock()
		registryRequests = append(registryRequests, r.Method+" "+r.URL.RequestURI())
		registryMu.Unlock()
		http.Error(w, "offline-registry-trap", http.StatusServiceUnavailable)
	}))
	defer registry.Close()

	cfgDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cfgDir, ".npmrc"), []byte(fmt.Sprintf("registry=%s/\nfetch-retries=0\nfetch-timeout=1000\n", registry.URL)), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEA_PLUGIN_BUNDLE", bundle)
	t.Setenv("OPENCODE_BIN", openCodeBin)

	if err := writePluginConfig(cfgDir); err != nil {
		t.Fatalf("writePluginConfig: %v", err)
	}

	projectRoot := t.TempDir()
	adapter, cleanup, err := bootstrapRuntimeAt(cfgDir, skill.SkillModeStrict, projectRoot)
	if err != nil {
		t.Fatalf("bootstrapRuntimeAt: %v", err)
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	agents, err := adapter.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents with offline registry trap: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("ListAgents returned no agents")
	}
	if hits := registryHits.Load(); hits != 0 {
		registryMu.Lock()
		requests := strings.Join(registryRequests, ", ")
		registryMu.Unlock()
		t.Fatalf("OpenCode attempted npm registry access during file plugin load: hits=%d requests=[%s]", hits, requests)
	}
}
