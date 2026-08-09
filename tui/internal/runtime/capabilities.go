package runtime

// RuntimeCapabilities declares the capabilities a Runtime supports.
type RuntimeCapabilities struct {
	Sessions           bool
	Streaming          bool
	Reasoning          bool
	FileRead           bool
	FileWrite          bool
	Edit               bool
	Bash               bool
	ToolApproval       bool
	Agents             bool
	Subagents          bool
	Skills             bool
	Plugins            bool
	Abort              bool
	MessageHistory     bool
	ContextCompaction  bool
}
