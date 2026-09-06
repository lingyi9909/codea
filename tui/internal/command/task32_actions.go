package command

// ActionModelCheck is the Task 32 local workspace action for bounded model
// qualification. It never routes through the ordinary user prompt path.
const ActionModelCheck Action = "model-check"

func init() {
	controlledBuiltinNames["model-check"] = struct{}{}
}
