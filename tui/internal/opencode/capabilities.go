package opencode

import "codea/tui/internal/runtime"

// OpenCodeCapabilities returns the verified V1 capability declaration for OpenCode v1.18.11.
func OpenCodeCapabilities() runtime.RuntimeCapabilities {
	return runtime.RuntimeCapabilities{
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
}
