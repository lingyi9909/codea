package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTestPluginBundle(t *testing.T) string {
	t.Helper()
	bundle := filepath.Join(t.TempDir(), "plugins", "index.js")
	if err := os.MkdirAll(filepath.Dir(bundle), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle, []byte("export default {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestWritePluginConfigPreservesExistingRuntimeConfig(t *testing.T) {
	bundle := writeTestPluginBundle(t)
	t.Setenv("CODEA_PLUGIN_BUNDLE", bundle)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "opencode.json")
	existing := `{
  "model": "private/model",
  "provider": {"private": {"name": "Private"}},
  "customField": {"enabled": true},
  "plugin": ["file:///old/plugin.js"]
}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writePluginConfig(cfgDir); err != nil {
		t.Fatalf("writePluginConfig: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["model"] != "private/model" {
		t.Fatalf("model lost: %#v", cfg)
	}
	if cfg["provider"] == nil {
		t.Fatalf("provider lost: %#v", cfg)
	}
	if cfg["customField"] == nil {
		t.Fatalf("customField lost: %#v", cfg)
	}
	plugins, ok := cfg["plugin"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("plugin not replaced with Codea bundle: %#v", cfg["plugin"])
	}
}

func TestWritePluginConfigInvalidJSONFailsClosed(t *testing.T) {
	bundle := writeTestPluginBundle(t)
	t.Setenv("CODEA_PLUGIN_BUNDLE", bundle)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "opencode.json")
	before := []byte(`{"model":"private/model",BROKEN}` + "\n")
	if err := os.WriteFile(cfgPath, before, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writePluginConfig(cfgDir); err == nil {
		t.Fatal("writePluginConfig must fail when existing opencode.json is invalid")
	}
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("invalid opencode.json was modified:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestMigratedC2ConfigSurvivesNormalPluginMaterialization(t *testing.T) {
	bundle := writeTestPluginBundle(t)
	t.Setenv("CODEA_PLUGIN_BUNDLE", bundle)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "opencode.json")
	migrated := `{
  "schemaVersion": 2,
  "model": "private/model",
  "provider": {"private": {"api": "http://model.internal"}},
  "migrationMarker": "c2"
}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(migrated), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writePluginConfig(cfgDir); err != nil {
		t.Fatalf("writePluginConfig: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["model"] != "private/model" || cfg["provider"] == nil {
		t.Fatalf("migrated C2 model/provider lost after plugin materialization: %#v", cfg)
	}
	if cfg["migrationMarker"] != "c2" {
		t.Fatalf("migrated C2 custom data lost: %#v", cfg)
	}
}
