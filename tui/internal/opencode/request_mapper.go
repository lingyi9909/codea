package opencode

import (
	"fmt"

	"codea/tui/internal/runtime"
)

// MappingError describes an unsupported or nil type during Domain-to-Vendor mapping.
type MappingError struct {
	Field string
	Type  string
}

func (e *MappingError) Error() string {
	return fmt.Sprintf("unsupported %s type: %s", e.Field, e.Type)
}

// MapCreateSessionRequest maps a Codea create-session request to an OpenCode session create request.
func MapCreateSessionRequest(req runtime.CreateSessionRequest) OpenCodeSessionCreateRequest {
	return OpenCodeSessionCreateRequest{
		Title: req.Title,
	}
}

// mapPromptAgent translates Codea-owned semantic agent names to the locked
// OpenCode vendor role. Codea's "general" is the main general-purpose chat
// agent; in OpenCode v1.18.11 the primary general-purpose agent is "build",
// while the vendor's own "general" name denotes a subagent. Keeping this alias
// here prevents vendor role names from leaking into Application/TUI code.
func mapPromptAgent(agent string) string {
	if agent == "general" {
		return "build"
	}
	return agent
}

// MapPromptRequest maps a Codea prompt request and session ID to an OpenCode prompt-async request.
func MapPromptRequest(sessionID runtime.SessionID, req runtime.PromptRequest) (string, OpenCodeSessionPromptAsyncRequest, error) {
	parts := make([]any, 0, len(req.Parts))
	for _, p := range req.Parts {
		mapped, err := mapPromptPart(p)
		if err != nil {
			return "", OpenCodeSessionPromptAsyncRequest{}, err
		}
		parts = append(parts, mapped)
	}

	var model *OpenCodeSessionPromptAsyncRequestModel
	if req.Model != nil {
		model = &OpenCodeSessionPromptAsyncRequestModel{
			ProviderID: req.Model.ProviderID,
			ModelID:    req.Model.ModelID,
		}
	}

	return string(sessionID), OpenCodeSessionPromptAsyncRequest{
		Agent:     mapPromptAgent(req.Agent),
		MessageID: req.MessageID,
		Model:     model,
		Parts:     parts,
	}, nil
}

func mapPromptPart(part runtime.PromptPart) (any, error) {
	if part == nil {
		return nil, &MappingError{Field: "PromptPart", Type: "nil"}
	}
	switch p := part.(type) {
	case runtime.TextPart:
		return OpenCodeTextPartInput{
			ID:        p.ID,
			Text:      p.Text,
			Synthetic: p.Synthetic,
			Ignored:   p.Ignored,
			Metadata:  p.Metadata,
			Type:      "text",
		}, nil
	case runtime.FilePart:
		source, err := mapFilePartSource(p.Source)
		if err != nil {
			return nil, err
		}
		return OpenCodeFilePartInput{
			ID:       p.ID,
			Filename: p.Filename,
			Mime:     p.MIME,
			Url:      p.URL,
			Source:   source,
			Type:     "file",
		}, nil
	case runtime.AgentPart:
		var source *OpenCodeAgentPartInputSource
		if p.Source != nil {
			source = &OpenCodeAgentPartInputSource{
				Start: p.Source.Start,
				End:   p.Source.End,
				Value: p.Source.Value,
			}
		}
		return OpenCodeAgentPartInput{
			ID:     p.ID,
			Name:   p.Name,
			Source: source,
			Type:   "agent",
		}, nil
	case runtime.SubtaskPart:
		var model *OpenCodeSubtaskPartInputModel
		if p.Model != nil {
			model = &OpenCodeSubtaskPartInputModel{
				ProviderID: p.Model.ProviderID,
				ModelID:    p.Model.ModelID,
			}
		}
		return OpenCodeSubtaskPartInput{
			ID:          p.ID,
			Agent:       p.Agent,
			Description: p.Description,
			Prompt:      p.Prompt,
			Command:     p.Command,
			Model:       model,
			Type:        "subtask",
		}, nil
	default:
		return nil, &MappingError{Field: "PromptPart", Type: fmt.Sprintf("%T", part)}
	}
}

func mapFilePartSource(source runtime.FilePartSource) (OpenCodeFilePartSource, error) {
	if source == nil {
		return nil, &MappingError{Field: "FilePartSource", Type: "nil"}
	}
	switch s := source.(type) {
	case runtime.FileSource:
		return OpenCodeFileSource{
			Type: s.Type,
			Path: s.Path,
			Text: mapFilePartSourceText(s.Text),
		}, nil
	case runtime.SymbolSource:
		return OpenCodeSymbolSource{
			Type: s.Type,
			Path: s.Path,
			Name: s.Name,
			Kind: int64(s.Kind),
			Text: mapFilePartSourceText(s.Text),
			Range: OpenCodeRange{
				Start: OpenCodeRangeStart{Line: int64(s.Range.Start.Line), Character: int64(s.Range.Start.Character)},
				End:   OpenCodeRangeEnd{Line: int64(s.Range.End.Line), Character: int64(s.Range.End.Character)},
			},
		}, nil
	case runtime.ResourceSource:
		return OpenCodeResourceSource{
			Type:       s.Type,
			ClientName: s.ClientName,
			Uri:        s.URI,
			Text:       mapFilePartSourceText(s.Text),
		}, nil
	default:
		return nil, &MappingError{Field: "FilePartSource", Type: fmt.Sprintf("%T", source)}
	}
}

func mapFilePartSourceText(text runtime.FilePartSourceText) OpenCodeFilePartSourceText {
	return OpenCodeFilePartSourceText{
		Start: text.Start,
		End:   text.End,
		Value: text.Value,
	}
}
