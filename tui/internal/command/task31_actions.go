package command

const (
	ActionCheckpoint  Action = "checkpoint"
	ActionCheckpoints Action = "checkpoints"
	ActionRestore     Action = "restore"
)

func init() {
	// Task 31 extends the controlled built-in namespace. Enterprise/project
	// command files cannot replace local checkpoint controls.
	controlledBuiltinNames["checkpoint"] = struct{}{}
	controlledBuiltinNames["checkpoints"] = struct{}{}
	controlledBuiltinNames["restore"] = struct{}{}
}
