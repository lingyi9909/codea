package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"codea/tui/internal/runtime"
)

// FactRef is a compact reference to evidence already collected by an
// enterprise agent. The handoff carries summaries/references, not a replay of
// the full conversation or raw tool payloads.
type FactRef struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// ToolResultRef identifies a completed tool result that General should reuse
// rather than executing again.
type ToolResultRef struct {
	ID      string `json:"id"`
	Tool    string `json:"tool"`
	Summary string `json:"summary"`
}

// AgentHandoff is the Codea-owned application payload used when an enterprise
// tool cannot continue safely and the user elects to continue with General.
// It deliberately keeps the same Runtime session and carries only structured
// continuation facts.
type AgentHandoff struct {
	SourceAgent    string            `json:"sourceAgent"`
	TargetAgent    string            `json:"targetAgent"`
	SessionID      runtime.SessionID `json:"sessionId"`
	UserGoal       string            `json:"userGoal"`
	TaskSummary    string            `json:"taskSummary"`
	CollectedFacts []FactRef         `json:"collectedFacts"`
	GeneratedFiles []string          `json:"generatedFiles"`
	ToolResults    []ToolResultRef   `json:"toolResults"`
	FailureReason  string            `json:"failureReason"`
}

type handoffPolicy struct {
	ReuseSession            bool `json:"reuseSession"`
	RecollectFacts          bool `json:"recollectFacts"`
	OverwriteGeneratedFiles bool `json:"overwriteGeneratedFiles"`
	ContinueOrExplain       bool `json:"continueOrExplain"`
}

type handoffEnvelope struct {
	SchemaVersion int          `json:"schemaVersion"`
	Kind          string       `json:"kind"`
	Handoff       AgentHandoff `json:"handoff"`
	Policy        handoffPolicy `json:"policy"`
}

// HandoffToGeneral continues an enterprise task through Codea's semantic
// General agent in the existing session. Validation is fail-closed so this
// path cannot be used as an arbitrary agent-selection privilege escalation.
func HandoffToGeneral(ctx context.Context, client runtime.AgentRuntime, handoff AgentHandoff) error {
	if client == nil {
		return fmt.Errorf("handoff runtime is required")
	}
	if strings.TrimSpace(handoff.SourceAgent) == "" {
		return fmt.Errorf("handoff sourceAgent is required")
	}
	if handoff.TargetAgent != "general" {
		return fmt.Errorf("handoff targetAgent must be general")
	}
	if strings.TrimSpace(string(handoff.SessionID)) == "" {
		return fmt.Errorf("handoff sessionId is required")
	}
	if strings.TrimSpace(handoff.UserGoal) == "" {
		return fmt.Errorf("handoff userGoal is required")
	}
	if strings.TrimSpace(handoff.FailureReason) == "" {
		return fmt.Errorf("handoff failureReason is required")
	}

	handoff.GeneratedFiles = uniqueNonEmpty(handoff.GeneratedFiles)
	envelope := handoffEnvelope{
		SchemaVersion: 1,
		Kind:          "agent_handoff",
		Handoff:       handoff,
		Policy: handoffPolicy{
			ReuseSession:            true,
			RecollectFacts:          false,
			OverwriteGeneratedFiles: false,
			ContinueOrExplain:       true,
		},
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal handoff: %w", err)
	}
	digest := sha256.Sum256(payload)
	messageID := "handoff-" + hex.EncodeToString(digest[:8])

	return client.Prompt(ctx, handoff.SessionID, runtime.PromptRequest{
		MessageID: messageID,
		Agent:     "general",
		Parts: []runtime.PromptPart{runtime.TextPart{
			Text:      string(payload),
			Synthetic: true,
		}},
	})
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
