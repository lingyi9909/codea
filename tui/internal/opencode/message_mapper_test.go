package opencode

import (
	"testing"

	"codea/tui/internal/runtime"
)

func TestMapSessionMessageConcatenatesTextParts(t *testing.T) {
	raw := map[string]any{
		"info": map[string]any{"id": "msg_1", "role": "assistant"},
		"parts": []any{
			map[string]any{"type": "text", "text": "Hello "},
			map[string]any{"type": "text", "text": "world"},
		},
	}

	got := MapSessionMessage(raw)

	want := runtime.Message{ID: "msg_1", Role: "assistant", Content: "Hello world"}
	if got != want {
		t.Errorf("MapSessionMessage = %+v, want %+v", got, want)
	}
}

func TestMapSessionMessageIgnoresNonTextParts(t *testing.T) {
	raw := map[string]any{
		"info": map[string]any{"id": "msg_2", "role": "user"},
		"parts": []any{
			map[string]any{"type": "tool", "name": "bash"},
			map[string]any{"type": "text", "text": "run this"},
			map[string]any{"type": "reasoning", "text": "thinking"},
		},
	}

	got := MapSessionMessage(raw)

	if got.Content != "run this" {
		t.Errorf("Content = %q, want %q (non-text parts must be ignored)", got.Content, "run this")
	}
	if got.Role != "user" {
		t.Errorf("Role = %q, want user", got.Role)
	}
}

func TestMapSessionMessageNonStructInputReturnsEmpty(t *testing.T) {
	if got := MapSessionMessage("not-an-object"); got != (runtime.Message{}) {
		t.Errorf("MapSessionMessage(non-object) = %+v, want zero Message", got)
	}
}
