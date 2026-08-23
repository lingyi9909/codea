package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codea/tui/internal/skill"
	"codea/tui/internal/update"
)

func TestPrepareCandidateConfigMaterializesResourcesAndPreservesModelConfig(t *testing.T) {
	versionDir := t.TempDir()
	configDir := t.TempDir()

	doctorWrite(t, filepath.Join(versionDir, "plugins", "index.js"), "export default {};\n")
	doctorWrite(t, filepath.Join(versionDir, "skills", "api-documentation", "SKILL.md"), "# api-documentation\n")
	doctorWrite(t, filepath.Join(versionDir, "agents", "api-documentation", "manifest.yaml"), "name: api-documentation\ndisplayName: API Documentation\ntools:\n  read: allow\n")
	doctorWrite(t, filepath.Join(versionDir, "agents", "api-documentation", "agent.md"), "candidate prompt\n")
	doctorWrite(t, filepath.Join(configDir, "opencode.json"), `{"model":"private/model","provider":{"private":{"name":"Private"}}}`+"\n")

	candidate := update.Candidate{Version: "2.0.0", VersionDir: versionDir, ConfigDir: configDir}
	if err := prepareCandidateConfig(candidate, skill.DefaultPolicy); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(configDir, "agents", "api-documentation.md")); err != nil {
		t.Fatalf("agent not materialized: %v", err)
	}
	if _, err := os.Stat(filepath.Join(configDir, "skills", "api-documentation", "SKILL.md")); err != nil {
		t.Fatalf("skill not materialized: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(configDir, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["model"] != "private/model" || cfg["provider"] == nil {
		t.Fatalf("existing model/provider config was lost: %#v", cfg)
	}
	plugins, ok := cfg["plugin"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("plugin config = %#v", cfg["plugin"])
	}
	if got, _ := plugins[0].(string); !strings.HasPrefix(got, "file:") || !strings.Contains(got, "plugins/index.js") {
		t.Fatalf("candidate plugin URL = %q", got)
	}
}

func TestCandidateRuntimeFactoryRejectsInvalidSkillMode(t *testing.T) {
	factory := NewCandidateRuntimeFactory(CandidateRuntimeOptions{SkillMode: skill.SkillMode("invalid")})
	_, _, _, err := factory.Start(t.Context(), update.Candidate{VersionDir: t.TempDir(), ConfigDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "invalid candidate skill mode") {
		t.Fatalf("err=%v, want invalid candidate skill mode", err)
	}
}
