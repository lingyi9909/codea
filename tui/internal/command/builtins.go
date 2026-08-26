package command

// BuiltinCommands returns the controlled command workspace surface implemented
// through Task 23. Professional-agent commands remain reserved for Task 24 and
// are intentionally not registered here.
func BuiltinCommands() []Definition {
	return []Definition{
		{Name: "help", Description: "Show available commands", Category: "workspace", Source: SourceBuiltin, Action: ActionHelp},
		{Name: "clear", Description: "Clear the visible conversation", Category: "workspace", Source: SourceBuiltin, Action: ActionClear},
		{Name: "status", Description: "Show current workspace status", Category: "workspace", Source: SourceBuiltin, Action: ActionStatus},
		{Name: "sessions", Description: "Open session workspace", Category: "workspace", Source: SourceBuiltin, Action: ActionSessions},
		{Name: "skills", Description: "Open skill workspace", Category: "workspace", Source: SourceBuiltin, Action: ActionSkills},
		{Name: "agents", Description: "List available runtime agents", Category: "runtime", Source: SourceBuiltin, Action: ActionAgents},
		{Name: "model", Description: "Select the model for the current session", Category: "runtime", Source: SourceBuiltin, Action: ActionModel},
		{Name: "compact", Description: "Compact the current session context", Category: "runtime", Source: SourceBuiltin, Action: ActionCompact, RequiredCapability: "context_compaction"},
		{Name: "cancel", Description: "Cancel the current response", Category: "runtime", Source: SourceBuiltin, Action: ActionCancel},
		{Name: "doctor", Description: "Run the shared Codea Doctor", Category: "runtime", Source: SourceBuiltin, Action: ActionDoctor},
	}
}
