package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func assertOfflinePluginSentinel(t *testing.T, cfgDir string) {
	t.Helper()
	if st, err := os.Stat(filepath.Join(cfgDir, "node_modules")); err != nil || !st.IsDir() {
		t.Fatalf("node_modules sentinel missing in %s: stat=%v err=%v", cfgDir, st, err)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		LockfileVersion int `json:"lockfileVersion"`
		Packages map[string]struct {
			Dependencies map[string]string `json:"dependencies"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.LockfileVersion != 3 {
		t.Fatalf("lockfileVersion=%d, want 3", lock.LockfileVersion)
	}
	root, ok := lock.Packages[""]
	if !ok {
		t.Fatal("package-lock root package missing")
	}
	if got := root.Dependencies["@opencode-ai/plugin"]; got != "1.18.11" {
		t.Fatalf("@opencode-ai/plugin=%q, want 1.18.11", got)
	}
}

func TestPrepareOfflinePluginDependencySeedsOpenCodeInstallSentinel(t *testing.T) {
	cfgDir := t.TempDir()
	if err := PrepareOfflinePluginDependency(cfgDir, "1.18.11"); err != nil {
		t.Fatalf("PrepareOfflinePluginDependency: %v", err)
	}
	assertOfflinePluginSentinel(t, cfgDir)
}

func TestPrepareOfflinePluginDependencyIsIdempotentOnWindows(t *testing.T) {
	cfgDir := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := PrepareOfflinePluginDependency(cfgDir, "1.18.11"); err != nil {
			t.Fatalf("PrepareOfflinePluginDependency call %d: %v", i+1, err)
		}
	}
	assertOfflinePluginSentinel(t, cfgDir)
}

func TestPrepareOfflinePluginDependencyPreservesExistingRootDependencies(t *testing.T) {
	cfgDir := t.TempDir()
	before := `{
  "name": "existing",
  "lockfileVersion": 3,
  "packages": {
    "": {
      "dependencies": {
        "existing-package": "2.3.4"
      }
    }
  }
}` + "\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "package-lock.json"), []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := PrepareOfflinePluginDependency(cfgDir, "1.18.11"); err != nil {
		t.Fatalf("PrepareOfflinePluginDependency: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Packages map[string]struct {
			Dependencies map[string]string `json:"dependencies"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatal(err)
	}
	if got := lock.Packages[""].Dependencies["existing-package"]; got != "2.3.4" {
		t.Fatalf("existing dependency changed: %q", got)
	}
	assertOfflinePluginSentinel(t, cfgDir)
}

func TestMergePluginConfigSeedsBothOpenCodeDependencyDirectories(t *testing.T) {
	cfgDir := t.TempDir()
	bundle := filepath.Join(t.TempDir(), "index.js")
	if err := os.WriteFile(bundle, []byte("export default {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MergePluginConfig(cfgDir, bundle, 0o644); err != nil {
		t.Fatalf("MergePluginConfig: %v", err)
	}
	// OpenCode v1.18.11 always installs dependencies in Global.Path.config
	// ($XDG_CONFIG_HOME/opencode) and also in OPENCODE_CONFIG_DIR. Codea sets
	// XDG_CONFIG_HOME to <cfgDir>/xdg/config, so both Codea-owned locations must
	// be pre-seeded or /agent can still wait on an external npm request.
	assertOfflinePluginSentinel(t, cfgDir)
	assertOfflinePluginSentinel(t, filepath.Join(cfgDir, "xdg", "config", "opencode"))
}
