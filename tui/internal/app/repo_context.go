package app

import (
	"context"
	"strings"

	"codea/tui/internal/repoctx"
	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

const repoContextPromptBudget = 8000

// RepoContextService is the narrow Codea-owned Application dependency used to
// build task-specific repository context. It deliberately exposes no Runtime or
// vendor-specific types.
type RepoContextService interface {
	BuildMap(context.Context, repoctx.Query) (repoctx.RepoMap, error)
}

type repoPromptIntent struct {
	request     runtime.PromptRequest
	displayText string
	promptText  string
	queryText   string
}

type repoContextResultMsg struct {
	intent repoPromptIntent
	mapOut repoctx.RepoMap
	err    error
}

// SetRepoContextService injects the repository context service created by the
// composition root for the already-resolved project directory.
func (m *Model) SetRepoContextService(service RepoContextService) {
	m.repoContextService = service
}

// RepoContextCmd performs all repository scanning outside Bubble Tea Update.
// The result is returned as a message so Model mutation stays single-threaded.
func RepoContextCmd(service RepoContextService, intent repoPromptIntent) tea.Cmd {
	return func() tea.Msg {
		if service == nil {
			return repoContextResultMsg{intent: intent}
		}
		q := repoctx.Query{
			Text:     strings.TrimSpace(intent.queryText),
			MaxChars: repoContextPromptBudget,
		}
		result, err := service.BuildMap(context.Background(), q)
		return repoContextResultMsg{intent: intent, mapOut: result, err: err}
	}
}

func buildRepoAwarePrompt(intent repoPromptIntent, repoMap repoctx.RepoMap, repoErr error) runtime.PromptRequest {
	req := intent.request
	parts := make([]runtime.PromptPart, 0, 3)
	if repoErr == nil {
		rendered := strings.TrimSpace(repoMap.Render())
		if rendered != "" {
			parts = append(parts, runtime.TextPart{
				Text:      rendered,
				Synthetic: true,
				Metadata: map[string]any{
					"codea.kind": "repo-map",
				},
			})
		}
	}
	if strategy, ok := taskStrategyPart(req.Agent); ok {
		parts = append(parts, strategy)
	}
	parts = append(parts, runtime.TextPart{Text: intent.promptText})
	req.Parts = parts
	return req
}

func (m *Model) handleRepoContextResult(msg repoContextResultMsg) tea.Cmd {
	if msg.err != nil {
		m.appendInfo("Repo Context unavailable; continuing with the original prompt: " + msg.err.Error())
	}
	req := buildRepoAwarePrompt(msg.intent, msg.mapOut, msg.err)
	if m.sessionID == "" {
		m.pendingPrompt = &req
		return CreateSessionCmd(m.runtimeClient, strings.TrimSpace(msg.intent.displayText))
	}
	return PromptCmd(m.runtimeClient, m.sessionID, req)
}
