package capability

import (
	"testing"

	"codea/tui/internal/runtime"
)

func TestCompareAllSupported(t *testing.T) {
	inv := &Inventory{Requirements: []CapabilityRequirement{
		{Name: "sessions", Level: Required},
		{Name: "streaming", Level: Required},
		{Name: "skills", Level: Optional},
		{Name: "futureFeature", Level: Deferred},
	}}

	caps := runtime.RuntimeCapabilities{
		Sessions:  true,
		Streaming: true,
		Skills:    true,
	}

	result := inv.Compare(caps)

	if len(result.RequiredMissing) != 0 {
		t.Errorf("no required should be missing, got %v", result.RequiredMissing)
	}
	if len(result.RequiredSupported) != 2 {
		t.Errorf("expected 2 required supported, got %d", len(result.RequiredSupported))
	}
	if len(result.OptionalMissing) != 0 {
		t.Errorf("no optional should be missing, got %v", result.OptionalMissing)
	}
	if len(result.OptionalSupported) != 1 {
		t.Errorf("expected 1 optional supported, got %d", len(result.OptionalSupported))
	}
	if len(result.Deferred) != 1 {
		t.Errorf("expected 1 deferred, got %d", len(result.Deferred))
	}
	if result.HasRequiredFailures() {
		t.Error("should not have required failures")
	}
}

func TestCompareRequiredMissing(t *testing.T) {
	inv := &Inventory{Requirements: []CapabilityRequirement{
		{Name: "sessions", Level: Required},
		{Name: "streaming", Level: Required},
	}}

	caps := runtime.RuntimeCapabilities{
		Sessions:  true,
		Streaming: false,
	}

	result := inv.Compare(caps)

	if len(result.RequiredSupported) != 1 {
		t.Errorf("expected 1 required supported, got %d", len(result.RequiredSupported))
	}
	if len(result.RequiredMissing) != 1 {
		t.Errorf("expected 1 required missing, got %d", len(result.RequiredMissing))
	}
	if result.RequiredMissing[0] != "streaming" {
		t.Errorf("streaming should be missing, got %v", result.RequiredMissing)
	}
	if !result.HasRequiredFailures() {
		t.Error("should have required failures")
	}
}

func TestCompareOptionalMissing(t *testing.T) {
	inv := &Inventory{Requirements: []CapabilityRequirement{
		{Name: "sessions", Level: Required},
		{Name: "skills", Level: Optional},
	}}

	caps := runtime.RuntimeCapabilities{
		Sessions: true,
		Skills:   false,
	}

	result := inv.Compare(caps)

	if len(result.RequiredMissing) != 0 {
		t.Errorf("no required should be missing, got %v", result.RequiredMissing)
	}
	if len(result.OptionalMissing) != 1 {
		t.Errorf("expected 1 optional missing, got %d", len(result.OptionalMissing))
	}
	// Optional missing should NOT block.
	if result.HasRequiredFailures() {
		t.Error("optional missing should not cause required failures")
	}
}

func TestCompareDeferredNotInGate(t *testing.T) {
	inv := &Inventory{Requirements: []CapabilityRequirement{
		{Name: "futureFeature", Level: Deferred},
	}}

	caps := runtime.RuntimeCapabilities{}

	result := inv.Compare(caps)

	if len(result.Deferred) != 1 {
		t.Errorf("expected 1 deferred, got %d", len(result.Deferred))
	}
	// Deferred capabilities never appear in missing lists.
	if len(result.RequiredMissing) != 0 {
		t.Errorf("deferred should not be in required missing, got %v", result.RequiredMissing)
	}
	if result.HasRequiredFailures() {
		t.Error("deferred should not cause required failures")
	}
}

func TestCompareUnknownCapability(t *testing.T) {
	inv := &Inventory{Requirements: []CapabilityRequirement{
		{Name: "someUnknownCap", Level: Required},
	}}

	caps := runtime.RuntimeCapabilities{}

	result := inv.Compare(caps)

	if len(result.RequiredMissing) != 1 {
		t.Errorf("unknown required capability should be missing, got %v", result.RequiredMissing)
	}
	if !result.HasRequiredFailures() {
		t.Error("unknown required capability should cause failure")
	}
}

func TestCompareAllFieldsMapped(t *testing.T) {
	// Verify every RuntimeCapabilities field has a corresponding mapping.
	inv := &Inventory{Requirements: []CapabilityRequirement{
		{Name: "sessions", Level: Required},
		{Name: "streaming", Level: Required},
		{Name: "reasoning", Level: Required},
		{Name: "fileRead", Level: Required},
		{Name: "fileWrite", Level: Required},
		{Name: "edit", Level: Required},
		{Name: "bash", Level: Required},
		{Name: "toolApproval", Level: Required},
		{Name: "agents", Level: Required},
		{Name: "subagents", Level: Required},
		{Name: "skills", Level: Required},
		{Name: "plugins", Level: Required},
		{Name: "abort", Level: Required},
		{Name: "messageHistory", Level: Required},
		{Name: "contextCompaction", Level: Required},
	}}

	caps := runtime.RuntimeCapabilities{
		Sessions:          true,
		Streaming:         true,
		Reasoning:         true,
		FileRead:          true,
		FileWrite:         true,
		Edit:              true,
		Bash:              true,
		ToolApproval:      true,
		Agents:            true,
		Subagents:         true,
		Skills:            true,
		Plugins:           true,
		Abort:             true,
		MessageHistory:    true,
		ContextCompaction: true,
	}

	result := inv.Compare(caps)

	if len(result.RequiredMissing) != 0 {
		t.Errorf("all required should be supported, missing: %v", result.RequiredMissing)
	}
	if len(result.RequiredSupported) != 15 {
		t.Errorf("expected 15 supported, got %d", len(result.RequiredSupported))
	}
}

func TestLoadAndCompareRealCapabilitiesYAML(t *testing.T) {
	inv, err := Load("../../../runtime/capabilities.yaml")
	if err != nil {
		t.Fatalf("Load of real capabilities.yaml failed: %v", err)
	}

	caps := runtime.RuntimeCapabilities{
		Sessions:          true,
		Streaming:         true,
		Reasoning:         true,
		FileRead:          true,
		FileWrite:         true,
		Edit:              true,
		Bash:              true,
		ToolApproval:      true,
		Agents:            true,
		Subagents:         true,
		Skills:            true,
		Plugins:           true,
		Abort:             true,
		MessageHistory:    true,
		ContextCompaction: true,
	}

	result := inv.Compare(caps)
	if result.HasRequiredFailures() {
		t.Errorf("real capabilities.yaml with all caps should pass, missing: %v", result.RequiredMissing)
	}
	if len(result.RequiredSupported) != 15 {
		t.Errorf("expected 15 required supported, got %d", len(result.RequiredSupported))
	}
}
