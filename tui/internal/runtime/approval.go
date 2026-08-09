package runtime

// ApprovalDecision is a permission reply decision.
type ApprovalDecision string

const (
	ApprovalOnce   ApprovalDecision = "once"
	ApprovalAlways ApprovalDecision = "always"
	ApprovalReject ApprovalDecision = "reject"
)

// ApprovalReply carries a decision and optional message for a permission reply.
type ApprovalReply struct {
	Decision ApprovalDecision
	Message  string
}
