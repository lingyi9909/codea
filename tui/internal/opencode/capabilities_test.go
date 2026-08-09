package opencode

import (
	"testing"

	"codea/tui/internal/runtime"
)

func TestOpenCodeCapabilitiesDeclaresAllV1Keys(t *testing.T) {
	caps := OpenCodeCapabilities()

	if !caps.Sessions {
		t.Fatal("expected Sessions=true")
	}
	if !caps.Streaming {
		t.Fatal("expected Streaming=true")
	}
	if !caps.Reasoning {
		t.Fatal("expected Reasoning=true")
	}
	if !caps.FileRead {
		t.Fatal("expected FileRead=true")
	}
	if !caps.FileWrite {
		t.Fatal("expected FileWrite=true")
	}
	if !caps.Edit {
		t.Fatal("expected Edit=true")
	}
	if !caps.Bash {
		t.Fatal("expected Bash=true")
	}
	if !caps.ToolApproval {
		t.Fatal("expected ToolApproval=true")
	}
	if !caps.Agents {
		t.Fatal("expected Agents=true")
	}
	if !caps.Subagents {
		t.Fatal("expected Subagents=true")
	}
	if !caps.Skills {
		t.Fatal("expected Skills=true")
	}
	if !caps.Plugins {
		t.Fatal("expected Plugins=true")
	}
	if !caps.Abort {
		t.Fatal("expected Abort=true")
	}
	if !caps.MessageHistory {
		t.Fatal("expected MessageHistory=true")
	}
	if !caps.ContextCompaction {
		t.Fatal("expected ContextCompaction=true")
	}
}

func TestOpenCodeCapabilitiesReturnsRuntimeCapabilities(t *testing.T) {
	var caps runtime.RuntimeCapabilities = OpenCodeCapabilities()
	_ = caps
}
