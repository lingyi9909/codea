package opencode

import (
	"fmt"

	"codea/tui/internal/runtime"
)

// MapCreateSessionRequest maps a Codea create-session request to an OpenCode session create request.
func MapCreateSessionRequest(req runtime.CreateSessionRequest) OpenCodeSessionCreateRequest {
	return OpenCodeSessionCreateRequest{
		Title: req.Title,
	}
}

// MapPromptRequest maps a Codea prompt request and session ID to an OpenCode prompt-async request.
func MapPromptRequest(sessionID runtime.SessionID, req runtime.PromptRequest) (string, OpenCodeSessionPromptAsyncRequest) {
	parts := make([]any, 0, len(req.Parts))
	for _, p := range req.Parts {
		parts = append(parts, mapPromptPart(p))
	}

	var model *OpenCodeSessionPromptAsyncRequestModel
	if req.Model != nil {
		model = &OpenCodeSessionPromptAsyncRequestModel{
			ProviderID: req.Model.ProviderID,
			ModelID:    req.Model.ModelID,
		}
	}

	return string(sessionID), OpenCodeSessionPromptAsyncRequest{
		Agent:     req.Agent,
		MessageID: req.MessageID,
		Model:     model,
		Parts:     parts,
	}
}

func mapPromptPart(part runtime.PromptPart) any {
	switch p := part.(type) {
	case runtime.TextPart:
		return OpenCodeTextPartInput{
			ID:        p.ID,
			Text:      p.Text,
			Synthetic: p.Synthetic,
			Ignored:   p.Ignored,
			Metadata:  p.Metadata,
			Type:      "text",
		}
	case runtime.FilePart:
		return OpenCodeFilePartInput{
			ID:       p.ID,
			Filename: p.Filename,
			Mime:     p.MIME,
			Url:      p.URL,
			Source:   mapFilePartSource(p.Source),
			Type:     "file",
		}
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
		}
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
		}
	default:
		panic(fmt.Sprintf("unsupported prompt part type: %T", part))
	}
}

func mapFilePartSource(source runtime.FilePartSource) OpenCodeFilePartSource {
	switch s := source.(type) {
	case runtime.FileSource:
		return OpenCodeFileSource{
			Type: s.Type,
			Path: s.Path,
			Text: mapFilePartSourceText(s.Text),
		}
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
		}
	case runtime.ResourceSource:
		return OpenCodeResourceSource{
			Type:       s.Type,
			ClientName: s.ClientName,
			Uri:        s.URI,
			Text:       mapFilePartSourceText(s.Text),
		}
	default:
		panic(fmt.Sprintf("unsupported file part source type: %T", source))
	}
}

func mapFilePartSourceText(text runtime.FilePartSourceText) OpenCodeFilePartSourceText {
	return OpenCodeFilePartSourceText{
		Start: text.Start,
		End:   text.End,
		Value: text.Value,
	}
}
