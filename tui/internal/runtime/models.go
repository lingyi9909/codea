package runtime

import "time"

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
	ID        string
	Title     string
	UpdatedAt time.Time
}

// Message is a Codea-owned conversation message, used for session-history
// rehydration. It carries only role + text content; vendor part DTOs are
// flattened by the adapter and never cross into the Application.
type Message struct {
	ID      string
	Role    string // "user" | "assistant"
	Content string
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
	Source   FilePartSource
}

func (FilePart) isPromptPart() {}

// FilePartSource is a sealed interface over the three file part source variants.
type FilePartSource interface {
	isFilePartSource()
}

// FilePartSourceText is a text selection within a file.
type FilePartSourceText struct {
	Start float64
	End   float64
	Value string
}

// FileSource is a file-based part source.
type FileSource struct {
	Type string
	Path string
	Text FilePartSourceText
}

func (FileSource) isFilePartSource() {}

// SymbolSource is a symbol-based part source.
type SymbolSource struct {
	Type  string
	Path  string
	Text  FilePartSourceText
	Range SymbolRange
	Name  string
	Kind  int
}

func (SymbolSource) isFilePartSource() {}

// ResourceSource is a resource-based part source.
type ResourceSource struct {
	Type       string
	Text       FilePartSourceText
	ClientName string
	URI        string
}

func (ResourceSource) isFilePartSource() {}

// SymbolRange is a line+character range in a file.
type SymbolRange struct {
	Start Position
	End   Position
}

// Position is a line and character offset.
type Position struct {
	Line      int
	Character int
}

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
