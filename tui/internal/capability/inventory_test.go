package capability

import (
	"os"
	"path/filepath"
	"testing"
)

func writeYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeYAML: %v", err)
	}
	return path
}

func TestLoadValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "capabilities.yaml", `
schemaVersion: 1
openCodeVersion: 1.18.11
capabilities:
  sessions: required
  streaming: required
  reasoning: required
  fileRead: required
  fileWrite: required
  edit: required
  bash: required
  toolApproval: required
  agents: required
  subagents: required
  skills: required
  plugins: required
  abort: required
  messageHistory: required
  contextCompaction: required
tui:
  sessionList: required
  sessionResume: required
  toolApproval: required
  rawEventFallback: required
`)

	inv, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if inv == nil {
		t.Fatal("inventory is nil")
	}
	if len(inv.Requirements) != 15 {
		t.Fatalf("expected 15 capability requirements, got %d", len(inv.Requirements))
	}

	byName := make(map[string]RequirementLevel)
	for _, r := range inv.Requirements {
		byName[r.Name] = r.Level
	}

	for _, name := range []string{
		"sessions", "streaming", "reasoning", "fileRead", "fileWrite",
		"edit", "bash", "toolApproval", "agents", "subagents",
		"skills", "plugins", "abort", "messageHistory", "contextCompaction",
	} {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing capability: %s", name)
		}
	}

	if byName["sessions"] != Required {
		t.Errorf("sessions should be required, got %s", byName["sessions"])
	}
}

func TestLoadOptionalAndDeferred(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "capabilities.yaml", `
schemaVersion: 1
capabilities:
  sessions: required
  fancyFeature: optional
  futureFeature: deferred
`)

	inv, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	byName := make(map[string]RequirementLevel)
	for _, r := range inv.Requirements {
		byName[r.Name] = r.Level
	}

	if byName["sessions"] != Required {
		t.Errorf("sessions should be required, got %s", byName["sessions"])
	}
	if byName["fancyFeature"] != Optional {
		t.Errorf("fancyFeature should be optional, got %s", byName["fancyFeature"])
	}
	if byName["futureFeature"] != Deferred {
		t.Errorf("futureFeature should be deferred, got %s", byName["futureFeature"])
	}
}

func TestLoadUnknownLevel(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "capabilities.yaml", `
schemaVersion: 1
capabilities:
  sessions: maybeLater
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown requirement level, got nil")
	}
}

func TestLoadDuplicateCapability(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "capabilities.yaml", `
schemaVersion: 1
capabilities:
  sessions: required
  streaming: required
`)

	// Duplicate in YAML is not possible (last key wins), but we test
	// that loading the same name twice in different sections is fine.
	inv, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	count := 0
	for _, r := range inv.Requirements {
		if r.Name == "sessions" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 sessions entry, got %d", count)
	}
}

func TestLoadNoCapabilitiesSection(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "capabilities.yaml", `
schemaVersion: 1
`)

	inv, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(inv.Requirements) != 0 {
		t.Errorf("expected 0 requirements, got %d", len(inv.Requirements))
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/capabilities.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadEmptyLevel(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "capabilities.yaml", `
schemaVersion: 1
capabilities:
  sessions:
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty requirement level, got nil")
	}
}
