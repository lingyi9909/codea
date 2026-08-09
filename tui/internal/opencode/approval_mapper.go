package opencode

import "codea/tui/internal/runtime"

// MapApprovalReply maps a Codea ApprovalReply to an OpenCode permission reply request.
func MapApprovalReply(reply runtime.ApprovalReply) OpenCodePermissionReplyRequest {
	return OpenCodePermissionReplyRequest{
		Reply:   string(reply.Decision),
		Message: reply.Message,
	}
}
