package command

// BuiltinCommands returns Task 22's fixed command workspace surface. Later
// V1.1 tasks may register additional commands, but Task 22 intentionally does
// not pre-register professional-agent or model-workspace commands.
func BuiltinCommands() []Definition {
	return []Definition{
		{Name: "help", Description: "Show available commands", Category: "workspace", Source: SourceBuiltin, Action: ActionHelp},
		{Name: "clear", Description: "Clear the visible conversation", Category: "workspace", Source: SourceBuiltin, Action: ActionClear},
		{Name: "status", Description: "Show current workspace status", Category: "workspace", Source: SourceBuiltin, Action: ActionStatus},
		{Name: "sessions", Description: "Open session workspace", Category: "workspace", Source: SourceBuiltin, Action: ActionSessions},
		{Name: "skills", Description: "Open skill workspace", Category: "workspace", Source: SourceBuiltin, Action: ActionSkills},
		{Name: "agents", Description: "List available runtime agents", Category: "runtime", Source: SourceBuiltin, Action: ActionAgents},
		{Name: "cancel", Description: "Cancel the current response", Category: "runtime", Source: SourceBuiltin, Action: ActionCancel},
		{Name: "doctor", Description: "Run the runtime health quick check", Category: "runtime", Source: SourceBuiltin, Action: ActionDoctor},
	}
}
