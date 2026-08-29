package app

import (
	"strings"

	"codea/tui/internal/runtime"
)

const generalTaskStrategy = `For read-only/explanatory work, do not create a plan.
Before your first project mutation or project command execution, create task_plan.
Keep the plan bounded to 3–7 steps and update task_step with evidence.`

func taskStrategyPart(agent string) (runtime.TextPart, bool) {
	if strings.ToLower(strings.TrimSpace(agent)) != "general" {
		return runtime.TextPart{}, false
	}
	return runtime.TextPart{
		Text:      generalTaskStrategy,
		Synthetic: true,
		Metadata: map[string]any{
			"codea.kind": "task-strategy",
		},
	}, true
}
