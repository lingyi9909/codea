package main

import (
	"context"

	"codea/tui/internal/app"
	"codea/tui/internal/checkpoint"
)

func configureCheckpoint(model *app.Model, codeaHome, projectDir string) {
	service, err := checkpoint.NewService(context.Background(), codeaHome, projectDir, checkpoint.NewGitRunner())
	if err != nil {
		model.SetCheckpointUnavailable(err)
		return
	}
	model.SetCheckpointService(service)
}
