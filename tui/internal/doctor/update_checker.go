package doctor

import (
	"context"
	"fmt"
	"time"

	runtimedomain "codea/tui/internal/runtime"
	"codea/tui/internal/update"
)

type CandidateRuntimeFactory interface {
	Start(context.Context, update.Candidate) (runtimedomain.AgentRuntime, string, func(), error)
}

type UpdateChecker struct {
	Factory                 CandidateRuntimeFactory
	ExpectedOpenCodeVersion string
	Timeout                 time.Duration
}

var _ update.CandidateChecker = (*UpdateChecker)(nil)

func (c *UpdateChecker) Check(ctx context.Context, phase update.CheckPhase, candidate update.Candidate) error {
	if c == nil || c.Factory == nil {
		return fmt.Errorf("candidate runtime factory is required")
	}
	rt, runtimeURL, cleanup, err := c.Factory.Start(ctx, candidate)
	if err != nil {
		return fmt.Errorf("%s candidate runtime: %w", phase, err)
	}
	if cleanup != nil {
		defer cleanup()
	}
	svc, err := NewCandidateService(candidate, rt, runtimeURL, c.ExpectedOpenCodeVersion, c.Timeout)
	if err != nil {
		return err
	}
	report := svc.Run(ctx)
	if report.HasFailures() {
		return fmt.Errorf("%s candidate doctor failed:\n%s", phase, FormatText(report))
	}
	return nil
}
