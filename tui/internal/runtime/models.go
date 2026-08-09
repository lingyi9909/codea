package runtime

// SessionID identifies a Codea session.
type SessionID string

// ApprovalID identifies a permission request.
type ApprovalID string

// ModelRef identifies a model by provider and model ID.
type ModelRef struct {
	ProviderID string
	ModelID    string
}

// HealthInfo reports Runtime health.
type HealthInfo struct {
	Healthy bool
	Version string
}

// Session represents an active Runtime session.
type Session struct {
	ID string
}

// CreateSessionRequest is the input for creating a session.
type CreateSessionRequest struct {
	Title string
}

// PromptRequest carries a prompt to a session.
type PromptRequest struct {
	MessageID string
	Agent     string
	Model     *ModelRef
	Parts     []PromptPart
}

// PromptPart is a sealed interface over the four prompt part variants.
type PromptPart interface {
	isPromptPart()
}

// TextPart carries a text prompt part.
type TextPart struct {
	ID        string
	Text      string
	Synthetic bool
	Ignored   bool
	Metadata  map[string]any
}

func (TextPart) isPromptPart() {}

// FilePart carries a file reference prompt part.
type FilePart struct {
	ID       string
	MIME     string
	URL      string
	Filename string
	Source   any
}

func (FilePart) isPromptPart() {}

// AgentPart carries an agent reference prompt part.
type AgentPart struct {
	ID     string
	Name   string
	Source *AgentPartSource
}

func (AgentPart) isPromptPart() {}

// AgentPartSource is the optional source range for an AgentPart.
type AgentPartSource struct {
	Start int64
	End   int64
	Value string
}

// SubtaskPart carries a subtask prompt part.
type SubtaskPart struct {
	ID          string
	Agent       string
	Description string
	Prompt      string
	Command     string
	Model       *ModelRef
}

func (SubtaskPart) isPromptPart() {}

// Agent describes an available Runtime agent.
type Agent struct {
	Name string
	Mode string
}
