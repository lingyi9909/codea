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

	for _, root := range []string{
		cfgDir,
		filepath.Join(cfgDir, "xdg", "config", "opencode"),
	} {
		assertOfflinePluginDependencyRoot(t, root)
	}
}

func assertOfflinePluginDependencyRoot(t *testing.T, root string) {
	t.Helper()

	if st, err := os.Stat(filepath.Join(root, "node_modules")); err != nil || !st.IsDir() {
		t.Fatalf("node_modules sentinel missing under %s: stat=%v err=%v", root, st, err)
	}

	pkgData, err := os.ReadFile(filepath.Join(root, "package.json"))
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
		t.Fatalf("unexpected package.json under %s: %s", root, pkgData)
	}

	lockData, err := os.ReadFile(filepath.Join(root, "package-lock.json"))
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
		t.Fatalf("unexpected package-lock.json under %s: %s", root, lockData)
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
