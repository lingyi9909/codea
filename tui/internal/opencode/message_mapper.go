package opencode

import (
	"encoding/json"
	"strings"

	"codea/tui/internal/runtime"
)

// MapSessionMessage maps a raw OpenCode session message to a Codea runtime.Message.
// The OpenCode message endpoint returns {"info": {"id", "role", ...}, "parts":
// [{"type", "text", ...}]}; only the role and the concatenated text-part content
// are preserved. Tool/reasoning/other parts are ignored — history rehydration
// shows the conversation text, not the tool timeline.
func MapSessionMessage(raw any) runtime.Message {
	data, err := json.Marshal(raw)
	if err != nil {
		return runtime.Message{}
	}

	var msg struct {
		Info struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"info"`
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return runtime.Message{}
	}

	var texts []string
	for _, p := range msg.Parts {
		if p.Type == "text" && p.Text != "" {
			texts = append(texts, p.Text)
		}
	}

	return runtime.Message{
		ID:      msg.Info.ID,
		Role:    msg.Info.Role,
		Content: strings.Join(texts, ""),
	}
}
