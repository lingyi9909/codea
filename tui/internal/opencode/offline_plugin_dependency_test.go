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
		t.Fatalf("node_modules sentinel missing: stat=%v err=%v", st, err)
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

func TestMergePluginConfigSeedsOfflineOpenCodeInstallSentinel(t *testing.T) {
	cfgDir := t.TempDir()
	bundle := filepath.Join(t.TempDir(), "index.js")
	if err := os.WriteFile(bundle, []byte("export default {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MergePluginConfig(cfgDir, bundle, 0o644); err != nil {
		t.Fatalf("MergePluginConfig: %v", err)
	}
	assertOfflinePluginSentinel(t, cfgDir)
}
