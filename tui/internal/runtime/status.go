package runtime

// RuntimeStatus is the lifecycle state of the OpenCode Runtime process.
//
// Lifecycle is owned exclusively by the RuntimeSupervisor; AgentRuntime
// deliberately has no Start/Stop/Status methods.
type RuntimeStatus string

const (
	RuntimeStopped  RuntimeStatus = "stopped"
	RuntimeStarting RuntimeStatus = "starting"
	RuntimeHealthy  RuntimeStatus = "healthy"
	RuntimeStopping RuntimeStatus = "stopping"
	RuntimeCrashed  RuntimeStatus = "crashed"
)
