package parity_test

import (
	"testing"

	"codea/tui/internal/capability"
	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

// nativeToolToCapability maps each concrete General Agent native tool to the
// product capability that backs it in runtime/capabilities.yaml. Task 9 must
// prove that Codea layers (Adapter/EventMapper/TUI) do NOT replace or weaken
// these native OpenCode tools.
var nativeToolToCapability = map[string]string{
	"read":   "fileRead",
	"grep":   "fileRead",
	"glob":   "fileRead",
	"write":  "fileWrite",
	"edit":   "edit",
	"bash":   "bash",
	"agent":  "agents",
	"skill":  "skills",
	"plugin": "plugins",
	"abort":  "abort",
}

// TestGeneralAgentNativeToolsRequired verifies the full General Agent native
// capability surface is declared required in the product requirements AND
// satisfied by the real OpenCodeAdapter through the AgentRuntime interface.
func TestGeneralAgentNativeToolsRequired(t *testing.T) {
	inv, err := capability.Load("../../../runtime/capabilities.yaml")
	if err != nil {
		t.Fatalf("load capabilities.yaml: %v", err)
	}

	required := []string{
		"sessions", "streaming", "reasoning",
		"fileRead", "fileWrite", "edit", "bash",
		"toolApproval", "agents", "subagents",
		"skills", "plugins", "abort",
		"messageHistory", "contextCompaction",
	}

	byName := make(map[string]capability.RequirementLevel, len(inv.Requirements))
	for _, req := range inv.Requirements {
		byName[req.Name] = req.Level
	}

	for _, name := range required {
		lvl, ok := byName[name]
		if !ok {
			t.Errorf("required capability %q missing from capabilities.yaml", name)
			continue
		}
		if lvl != capability.Required {
			t.Errorf("capability %q is %q, want required", name, lvl)
		}
	}

	adapter := opencode.NewOpenCodeAdapter("http://127.0.0.1:1", "", "")
	var rt runtime.AgentRuntime = adapter
	result := inv.Compare(rt.Capabilities())
	if result.HasRequiredFailures() {
		t.Errorf("OpenCodeAdapter.Capabilities() missing required: %v", result.RequiredMissing)
	}
}

// TestNativeToolBackedByCapability proves every concrete native tool is backed
// by a required product capability — Codea does not silently strip any of them.
func TestNativeToolBackedByCapability(t *testing.T) {
	inv, err := capability.Load("../../../runtime/capabilities.yaml")
	if err != nil {
		t.Fatalf("load capabilities.yaml: %v", err)
	}
	byName := make(map[string]capability.RequirementLevel, len(inv.Requirements))
	for _, req := range inv.Requirements {
		byName[req.Name] = req.Level
	}

	for tool, capName := range nativeToolToCapability {
		lvl, ok := byName[capName]
		if !ok {
			t.Errorf("native tool %q → capability %q not declared", tool, capName)
			continue
		}
		if lvl != capability.Required {
			t.Errorf("native tool %q → capability %q is %q, want required", tool, capName, lvl)
		}
	}
}
