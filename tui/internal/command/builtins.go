package command

// BuiltinCommands returns the controlled V1.1 command workspace surface.
// Professional commands route directly to their fixed enterprise Agent through
// the same PromptRequest path as every other Codea prompt; they never delegate
// to the General Agent for route selection.
func BuiltinCommands() []Definition {
	return []Definition{
		{Name: "help", Description: "Show available commands", Category: "workspace", Source: SourceBuiltin, Action: ActionHelp},
		{Name: "clear", Description: "Clear the visible conversation", Category: "workspace", Source: SourceBuiltin, Action: ActionClear},
		{Name: "status", Description: "Show current workspace status", Category: "workspace", Source: SourceBuiltin, Action: ActionStatus},
		{Name: "sessions", Description: "Open session workspace", Category: "workspace", Source: SourceBuiltin, Action: ActionSessions},
		{Name: "skills", Description: "Open skill workspace", Category: "workspace", Source: SourceBuiltin, Action: ActionSkills},
		{Name: "agents", Description: "Select an available runtime agent", Category: "runtime", Source: SourceBuiltin, Action: ActionAgents},
		{Name: "model", Description: "Select the model for the current session", Category: "runtime", Source: SourceBuiltin, Action: ActionModel},
		{Name: "compact", Description: "Compact the current session context", Category: "runtime", Source: SourceBuiltin, Action: ActionCompact, RequiredCapability: "context_compaction"},
		{Name: "cancel", Description: "Cancel the current response", Category: "runtime", Source: SourceBuiltin, Action: ActionCancel},
		{Name: "doctor", Description: "Run the shared Codea Doctor", Category: "runtime", Source: SourceBuiltin, Action: ActionDoctor},
		{Name: "review", Description: "Review code with Code Reviewer", Category: "professional", Usage: "/review [target]", Source: SourceBuiltin, Action: ActionPrompt, Agent: "code-reviewer", Template: "$ARGUMENTS"},
		{Name: "test", Description: "Generate or repair unit tests", Category: "professional", Usage: "/test [target]", Source: SourceBuiltin, Action: ActionPrompt, Agent: "unit-test-generator", Template: "$ARGUMENTS"},
		{Name: "api-doc", Description: "Generate API documentation", Category: "professional", Usage: "/api-doc [target]", Source: SourceBuiltin, Action: ActionPrompt, Agent: "api-documentation", Template: "$ARGUMENTS"},
		{Name: "debug", Description: "Diagnose, fix, and re-verify a failure", Category: "professional", Usage: "/debug [failure evidence]", Source: SourceBuiltin, Action: ActionPrompt, Agent: "debug", Template: "$ARGUMENTS"},
	}
}
