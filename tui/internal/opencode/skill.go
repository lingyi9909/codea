package opencode

import (
	"context"
	"net/http"
	"net/url"

	"codea/tui/internal/runtime"
)

// ListSkills returns the skills the OpenCode runtime currently has loaded. A
// non-empty directory scopes the query so project-scoped skills are resolved
// against the correct workspace.
func (client *HTTPClient) ListSkills(ctx context.Context, directory string) (OpenCodeAppSkillsResponse, error) {
	path := "/skill"
	if directory != "" {
		path += "?directory=" + url.QueryEscape(directory)
	}
	var resp OpenCodeAppSkillsResponse
	if err := client.doJSON(ctx, http.MethodGet, path, nil, &resp, http.StatusOK); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListSkills implements runtime.SkillRuntime by querying the OpenCode /skill
// endpoint and mapping the raw vendor DTO to the Codea runtime contract.
func (a *OpenCodeAdapter) ListSkills(ctx context.Context, directory string) ([]runtime.LoadedSkill, error) {
	raw, err := a.httpClient.ListSkills(ctx, directory)
	if err != nil {
		return nil, classifyError("ListSkills", err)
	}
	result := make([]runtime.LoadedSkill, len(raw))
	for i, s := range raw {
		result[i] = runtime.LoadedSkill{
			Name:        s.Name,
			Description: s.Description,
			Location:    s.Location,
		}
	}
	return result, nil
}

var _ runtime.SkillRuntime = (*OpenCodeAdapter)(nil)
