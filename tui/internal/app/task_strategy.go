package app

import (
	"fmt"
	"strings"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/runtime"
)

const generalTaskStrategy = `For read-only/explanatory work, do not create a plan.
Before your first project mutation or project command execution, create task_plan.
Keep the machine-valid plan bounded to 3–7 steps and update task_step with evidence.`

func taskStrategyPart(agent string, strategy modelprofile.Strategy) (runtime.TextPart, bool) {
	if strings.ToLower(strings.TrimSpace(agent)) != "general" {
		return runtime.TextPart{}, false
	}
	preference := "Prefer 3–5 concise steps."
	switch strategy.Level {
	case modelprofile.CapabilityStrong:
		preference = "Prefer 3–7 steps as task complexity requires."
	case modelprofile.CapabilityWeak:
		preference = "Prefer exactly 3 concise steps when the task can be represented in three steps."
	}
	text := fmt.Sprintf("%s\nAdaptive planning preference: %s", generalTaskStrategy, preference)
	return runtime.TextPart{
		Text:      text,
		Synthetic: true,
		Metadata: map[string]any{
			"codea.kind":  "task-strategy",
			"codea.level": string(strategy.Level),
		},
	}, true
}
