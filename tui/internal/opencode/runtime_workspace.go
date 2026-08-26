package opencode

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"codea/tui/internal/runtime"
)

type providerListResponse struct {
	All []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Models map[string]struct {
			ID         string `json:"id"`
			ProviderID string `json:"providerID"`
			Name       string `json:"name"`
		} `json:"models"`
	} `json:"all"`
	Default   map[string]string `json:"default"`
	Connected []string          `json:"connected"`
}

type sessionModelEnvelope struct {
	Info struct {
		Role       string `json:"role"`
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
		Model      *struct {
			ProviderID string `json:"providerID"`
			ModelID    string `json:"modelID"`
		} `json:"model,omitempty"`
	} `json:"info"`
}

type summarizeSessionRequest struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

func (client *HTTPClient) listProviderModels(ctx context.Context) (providerListResponse, error) {
	var resp providerListResponse
	if err := client.doJSON(ctx, http.MethodGet, "/provider", nil, &resp, http.StatusOK); err != nil {
		return providerListResponse{}, err
	}
	return resp, nil
}

func (client *HTTPClient) lastSessionModel(ctx context.Context, sessionID string) (runtime.ModelRef, bool, error) {
	sp, err := pathSegment("session ID", sessionID)
	if err != nil {
		return runtime.ModelRef{}, false, err
	}
	var messages []sessionModelEnvelope
	if err := client.doJSON(ctx, http.MethodGet, "/session/"+sp+"/message", nil, &messages, http.StatusOK); err != nil {
		return runtime.ModelRef{}, false, err
	}
	for i := len(messages) - 1; i >= 0; i-- {
		info := messages[i].Info
		providerID, modelID := info.ProviderID, info.ModelID
		if info.Model != nil {
			if providerID == "" {
				providerID = info.Model.ProviderID
			}
			if modelID == "" {
				modelID = info.Model.ModelID
			}
		}
		if providerID != "" && modelID != "" {
			return runtime.ModelRef{ProviderID: providerID, ModelID: modelID}, true, nil
		}
	}
	return runtime.ModelRef{}, false, nil
}

func (client *HTTPClient) summarizeSession(ctx context.Context, sessionID string, model runtime.ModelRef) error {
	sp, err := pathSegment("session ID", sessionID)
	if err != nil {
		return err
	}
	var accepted bool
	if err := client.doJSON(ctx, http.MethodPost, "/session/"+sp+"/summarize", summarizeSessionRequest{
		ProviderID: model.ProviderID,
		ModelID:    model.ModelID,
	}, &accepted, http.StatusOK); err != nil {
		return err
	}
	if !accepted {
		return fmt.Errorf("session %s summarize was not accepted", sessionID)
	}
	return nil
}

// ListModels returns the real model set currently exposed by connected
// OpenCode providers. Only Codea-owned safe metadata crosses the adapter.
func (a *OpenCodeAdapter) ListModels(ctx context.Context) ([]runtime.Model, error) {
	providers, err := a.httpClient.listProviderModels(ctx)
	if err != nil {
		return nil, classifyError("ListModels", err)
	}
	connected := make(map[string]struct{}, len(providers.Connected))
	for _, id := range providers.Connected {
		connected[id] = struct{}{}
	}
	result := make([]runtime.Model, 0)
	for _, provider := range providers.All {
		if _, ok := connected[provider.ID]; !ok {
			continue
		}
		for key, model := range provider.Models {
			modelID := model.ID
			if modelID == "" {
				modelID = key
			}
			providerID := model.ProviderID
			if providerID == "" {
				providerID = provider.ID
			}
			if providerID == "" || modelID == "" {
				continue
			}
			name := model.Name
			if name == "" {
				name = modelID
			}
			result = append(result, runtime.Model{
				Ref:          runtime.ModelRef{ProviderID: providerID, ModelID: modelID},
				Name:         name,
				ProviderName: provider.Name,
				Default:      providers.Default[providerID] == modelID,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Ref.ProviderID != result[j].Ref.ProviderID {
			return result[i].Ref.ProviderID < result[j].Ref.ProviderID
		}
		return result[i].Ref.ModelID < result[j].Ref.ModelID
	})
	return result, nil
}

// CompactSession performs OpenCode's real same-session summarize operation.
// v1.18.11 requires the current provider/model, so the adapter resolves it from
// session history and fails closed when the Runtime has not provided evidence.
func (a *OpenCodeAdapter) CompactSession(ctx context.Context, sessionID runtime.SessionID) error {
	if !a.Capabilities().ContextCompaction {
		return runtime.NewIncompatibleError("CompactSession", "runtime does not support context compaction")
	}
	model, ok, err := a.httpClient.lastSessionModel(ctx, string(sessionID))
	if err != nil {
		return classifyError("CompactSession", err)
	}
	if !ok {
		return runtime.NewIncompatibleError("CompactSession", "cannot determine current session model for compaction")
	}
	if err := a.httpClient.summarizeSession(ctx, string(sessionID), model); err != nil {
		return classifyError("CompactSession", err)
	}
	return nil
}
