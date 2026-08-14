package opencode

import (
	"time"

	"codea/tui/internal/runtime"
)

// MapSession maps an OpenCode session info DTO to the Codea session domain.
// The OpenCode DTO never crosses into the Application; only this mapped value
// does.
func MapSession(info OpenCodeSessionV2Info) runtime.Session {
	return runtime.Session{
		ID:        info.ID,
		Title:     info.Title,
		UpdatedAt: time.UnixMilli(int64(info.Time.Updated)),
	}
}
