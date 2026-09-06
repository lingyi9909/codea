package main

import (
	"context"

	"codea/tui/internal/app"
	"codea/tui/internal/checkpoint"
	"codea/tui/internal/modelprofile"
)

func configureCheckpoint(model *app.Model, codeaHome, projectDir string) {
	// Task 32 model profiles are independent of Git checkpoint availability.
	// Configure their local safe store before initializing the optional shadow
	// Git service so a missing Git binary cannot disable /model-check.
	model.SetModelProfileStore(modelprofile.NewStore(codeaHome))

	service, err := checkpoint.NewService(context.Background(), codeaHome, projectDir, checkpoint.NewGitRunner())
	if err != nil {
		model.SetCheckpointUnavailable(err)
		return
	}
	model.SetCheckpointService(service)
}
