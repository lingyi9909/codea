package runtime

import "context"

// AgentRuntime is the Codea-owned contract that every Runtime adapter must implement.
type AgentRuntime interface {
	Health(ctx context.Context) (HealthInfo, error)
	CreateSession(ctx context.Context, req CreateSessionRequest) (Session, error)
	Prompt(ctx context.Context, sessionID SessionID, req PromptRequest) error
	Subscribe(ctx context.Context) (<-chan Event, error)
	ReplyApproval(ctx context.Context, approvalID ApprovalID, reply ApprovalReply) error
	Cancel(ctx context.Context, sessionID SessionID) error
	ListAgents(ctx context.Context) ([]Agent, error)
	ListModels(ctx context.Context) ([]Model, error)
	ListSessions(ctx context.Context) ([]Session, error)
	GetSessionMessages(ctx context.Context, sessionID SessionID) ([]Message, error)
	CompactSession(ctx context.Context, sessionID SessionID) error
	Capabilities() RuntimeCapabilities
}
