package app

import (
	"context"
	"strings"

	"codea/tui/internal/modelprofile"
	"codea/tui/internal/repoctx"
	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

const repoContextPromptBudget = 8000

// RepoContextService is the narrow Codea-owned Application dependency used to
// build task-specific repository context. It deliberately exposes no Runtime or
// vendor-specific types. Invalidate is used after a checkpoint restore so any
// future cached/indexed implementation cannot serve stale source structure.
type RepoContextService interface {
	BuildMap(context.Context, repoctx.Query) (repoctx.RepoMap, error)
	Invalidate()
}

type repoPromptIntent struct {
	request     runtime.PromptRequest
	displayText string
	promptText  string
	queryText   string
	strategy    modelprofile.Strategy
}

type repoContextResultMsg struct {
	intent repoPromptIntent
	mapOut repoctx.RepoMap
	err    error
}

func (m *Model) SetRepoContextService(service RepoContextService) {
	m.repoContextService = service
}

func RepoContextCmd(service RepoContextService, intent repoPromptIntent) tea.Cmd {
	return func() tea.Msg {
		if service == nil {
			return repoContextResultMsg{intent: intent}
		}
		budget := intent.strategy.RepoMapMaxChars
		if budget <= 0 {
			budget = repoContextPromptBudget
		}
		q := repoctx.Query{Text: strings.TrimSpace(intent.queryText), MaxChars: budget}
		result, err := service.BuildMap(context.Background(), q)
		return repoContextResultMsg{intent: intent, mapOut: result, err: err}
	}
}

func buildRepoAwarePrompt(intent repoPromptIntent, repoMap repoctx.RepoMap, repoErr error) runtime.PromptRequest {
	req := intent.request
	strategy := intent.strategy
	if strategy.RepoMapMaxChars <= 0 {
		strategy = mediumModelStrategy()
	}
	parts := make([]runtime.PromptPart, 0, 4)
	parts = append(parts, modelStrategyPart(strategy))
	if repoErr == nil {
		rendered := strings.TrimSpace(repoMap.Render())
		if rendered != "" {
			parts = append(parts, runtime.TextPart{
				Text:      rendered,
				Synthetic: true,
				Metadata:  map[string]any{"codea.kind": "repo-map"},
			})
		}
	}
	if taskStrategy, ok := taskStrategyPart(req.Agent, strategy); ok {
		parts = append(parts, taskStrategy)
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
