package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareOfflinePluginDependenciesPreventsOpenCodeNetworkInstall(t *testing.T) {
	cfgDir := t.TempDir()

	if err := prepareOfflinePluginDependencies(cfgDir, "1.18.11"); err != nil {
		t.Fatalf("prepareOfflinePluginDependencies: %v", err)
	}

	if st, err := os.Stat(filepath.Join(cfgDir, "node_modules")); err != nil || !st.IsDir() {
		t.Fatalf("node_modules sentinel missing: stat=%v err=%v", st, err)
	}

	pkgData, err := os.ReadFile(filepath.Join(cfgDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Private      bool              `json:"private"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(pkgData, &pkg); err != nil {
		t.Fatal(err)
	}
	if !pkg.Private || pkg.Dependencies["@opencode-ai/plugin"] != "1.18.11" {
		t.Fatalf("unexpected package.json: %s", pkgData)
	}

	lockData, err := os.ReadFile(filepath.Join(cfgDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		LockfileVersion int `json:"lockfileVersion"`
		Packages        map[string]struct {
			Dependencies map[string]string `json:"dependencies"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(lockData, &lock); err != nil {
		t.Fatal(err)
	}
	if lock.LockfileVersion != 3 || lock.Packages[""].Dependencies["@opencode-ai/plugin"] != "1.18.11" {
		t.Fatalf("unexpected package-lock.json: %s", lockData)
	}
}

func TestPrepareOfflinePluginDependenciesRejectsVersionDrift(t *testing.T) {
	cfgDir := t.TempDir()
	if err := prepareOfflinePluginDependencies(cfgDir, "1.18.11"); err != nil {
		t.Fatal(err)
	}
	if err := prepareOfflinePluginDependencies(cfgDir, "1.18.12"); err == nil {
		t.Fatal("expected version drift to fail closed")
	}
}
