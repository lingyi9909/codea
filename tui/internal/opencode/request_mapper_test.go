package opencode

import (
	"encoding/json"
	"errors"
	"testing"

	"codea/tui/internal/runtime"
)

func TestMapCreateSessionRequestTitle(t *testing.T) {
	got := MapCreateSessionRequest(runtime.CreateSessionRequest{
		Title: "test session",
	})
	if got.Title != "test session" {
		t.Fatalf("expected Title=test session, got %q", got.Title)
	}
}

func TestMapCreateSessionRequestEmpty(t *testing.T) {
	got := MapCreateSessionRequest(runtime.CreateSessionRequest{})
	if got.Title != "" {
		t.Fatalf("expected empty title, got %q", got.Title)
	}
}

func TestMapPromptRequestWithTextPart(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		MessageID: "msg-1",
		Agent:     "general",
		Model:     &runtime.ModelRef{ProviderID: "deepseek", ModelID: "v3"},
		Parts: []runtime.PromptPart{
			runtime.TextPart{ID: "p1", Text: "hello", Synthetic: false, Ignored: false, Metadata: map[string]any{"key": "value"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.MessageID != "msg-1" {
		t.Fatalf("expected MessageID=msg-1, got %q", req.MessageID)
	}
	if req.Agent != "general" {
		t.Fatalf("expected Agent=general, got %q", req.Agent)
	}
	if req.Model == nil || req.Model.ProviderID != "deepseek" || req.Model.ModelID != "v3" {
		t.Fatalf("unexpected model: %+v", req.Model)
	}
	if len(req.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(req.Parts))
	}
}

func TestMapPromptRequestTextPartJSON(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.TextPart{ID: "p1", Text: "hello", Metadata: map[string]any{"k": "v"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "text" {
		t.Fatalf("expected discriminator type=text, got %v", m["type"])
	}
	if m["text"] != "hello" {
		t.Fatalf("expected text=hello, got %v", m["text"])
	}
}

func TestMapPromptRequestWithFilePartFileSource(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.FilePart{
				ID:       "f1",
				MIME:     "text/x-go",
				URL:      "file:///main.go",
				Filename: "main.go",
				Source: runtime.FileSource{
					Type: "file",
					Path: "/src/main.go",
					Text: runtime.FilePartSourceText{Start: 0, End: 10, Value: "package"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(req.Parts))
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "file" {
		t.Fatalf("expected discriminator type=file, got %v", m["type"])
	}
	if m["filename"] != "main.go" {
		t.Fatalf("expected filename=main.go, got %v", m["filename"])
	}
	if m["mime"] != "text/x-go" {
		t.Fatalf("expected mime=text/x-go, got %v", m["mime"])
	}
}

func TestMapPromptRequestWithFilePartSymbolSource(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.FilePart{
				ID:       "f2",
				MIME:     "text/x-go",
				Filename: "main.go",
				Source: runtime.SymbolSource{
					Type: "symbol",
					Path: "/src/main.go",
					Name: "MyFunc",
					Kind: 12,
					Text: runtime.FilePartSourceText{Start: 0, End: 20, Value: "func MyFunc()"},
					Range: runtime.SymbolRange{
						Start: runtime.Position{Line: 5, Character: 0},
						End:   runtime.Position{Line: 5, Character: 14},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	source, ok := m["source"].(map[string]any)
	if !ok {
		t.Fatal("source must be an object")
	}
	if source["type"] != "symbol" {
		t.Fatalf("expected source type=symbol, got %v", source["type"])
	}
	if source["name"] != "MyFunc" {
		t.Fatalf("expected source name=MyFunc, got %v", source["name"])
	}
}

func TestMapPromptRequestWithFilePartResourceSource(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.FilePart{
				ID:       "f3",
				MIME:     "text/plain",
				Filename: "res.txt",
				Source: runtime.ResourceSource{
					Type:       "resource",
					ClientName: "test-client",
					URI:        "res://data",
					Text:       runtime.FilePartSourceText{Start: 0, End: 5, Value: "data"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	source, ok := m["source"].(map[string]any)
	if !ok {
		t.Fatal("source must be an object")
	}
	if source["type"] != "resource" {
		t.Fatalf("expected source type=resource, got %v", source["type"])
	}
	if source["clientName"] != "test-client" {
		t.Fatalf("expected source clientName=test-client, got %v", source["clientName"])
	}
}

func TestMapPromptRequestWithAgentPart(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.AgentPart{
				ID:   "a1",
				Name: "reviewer",
				Source: &runtime.AgentPartSource{
					Start: 0,
					End:   100,
					Value: "some text",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "agent" {
		t.Fatalf("expected discriminator type=agent, got %v", m["type"])
	}
	if m["name"] != "reviewer" {
		t.Fatalf("expected name=reviewer, got %v", m["name"])
	}
}

func TestMapPromptRequestWithAgentPartNoSource(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.AgentPart{ID: "a2", Name: "general"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if _, hasSource := m["source"]; hasSource {
		t.Fatal("source must be omitted when nil")
	}
}

func TestMapPromptRequestWithSubtaskPart(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.SubtaskPart{
				ID:          "s1",
				Agent:       "code-reviewer",
				Description: "review the code",
				Prompt:      "please review",
				Command:     "review",
				Model:       &runtime.ModelRef{ProviderID: "openai", ModelID: "gpt-5"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "subtask" {
		t.Fatalf("expected discriminator type=subtask, got %v", m["type"])
	}
	if m["agent"] != "code-reviewer" {
		t.Fatalf("expected agent=code-reviewer, got %v", m["agent"])
	}
	if m["description"] != "review the code" {
		t.Fatalf("expected description, got %v", m["description"])
	}
	if m["prompt"] != "please review" {
		t.Fatalf("expected prompt, got %v", m["prompt"])
	}
	model, ok := m["model"].(map[string]any)
	if !ok {
		t.Fatal("model must be an object")
	}
	if model["providerID"] != "openai" || model["modelID"] != "gpt-5" {
		t.Fatalf("unexpected model: %+v", model)
	}
}

func TestMapPromptRequestWithSubtaskPartNoModel(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.SubtaskPart{
				ID:          "s2",
				Agent:       "reviewer",
				Description: "review",
				Prompt:      "review this",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(req.Parts[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if _, hasModel := m["model"]; hasModel {
		t.Fatal("model must be omitted when nil")
	}
}

func TestMapPromptRequestNilModel(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		MessageID: "msg-1",
		Model:     nil,
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: "hi"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Model != nil {
		t.Fatal("model must be nil when not provided")
	}
}

func TestMapPromptRequestAllParts(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.TextPart{Text: "t"},
			runtime.FilePart{MIME: "text/plain", Filename: "f.txt", Source: runtime.FileSource{Type: "file", Path: "/f.txt"}},
			runtime.AgentPart{Name: "a"},
			runtime.SubtaskPart{Agent: "s", Description: "d", Prompt: "p"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(req.Parts))
	}
}

func TestMapPromptRequestRejectsNilPart(t *testing.T) {
	_, _, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{nil},
	})
	if err == nil {
		t.Fatal("expected error for nil PromptPart")
	}
	var me *MappingError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MappingError, got %T: %v", err, err)
	}
	if me.Field != "PromptPart" || me.Type != "nil" {
		t.Fatalf("unexpected MappingError: Field=%q Type=%q", me.Field, me.Type)
	}
}

func TestMapPromptRequestRejectsNilFileSource(t *testing.T) {
	_, _, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.FilePart{MIME: "text/plain", Filename: "f.txt", Source: nil},
		},
	})
	if err == nil {
		t.Fatal("expected error for nil FilePartSource")
	}
	var me *MappingError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MappingError, got %T: %v", err, err)
	}
	if me.Field != "FilePartSource" || me.Type != "nil" {
		t.Fatalf("unexpected MappingError: Field=%q Type=%q", me.Field, me.Type)
	}
}

func TestMapPromptRequestRejectsNilPartStopsEarly(t *testing.T) {
	_, _, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Parts: []runtime.PromptPart{
			runtime.TextPart{Text: "ok"},
			nil,
			runtime.TextPart{Text: "after"},
		},
	})
	if err == nil {
		t.Fatal("expected error for nil PromptPart")
	}
	var me *MappingError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MappingError, got %T: %v", err, err)
	}
}
