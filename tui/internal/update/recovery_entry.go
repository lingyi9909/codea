package update

import (
	"context"
	"fmt"
)

type recoveryOnlyChecker struct{}

func (recoveryOnlyChecker) Check(context.Context, CheckPhase, Candidate) error {
	return fmt.Errorf("recovery-only service cannot perform an upgrade")
}

// RecoverHome is the narrow crash-recovery entry point used by codea doctor.
// It deliberately constructs a service whose checker always rejects Upgrade,
// so the recovery path cannot accidentally become a BasicChecker bypass.
func RecoverHome(ctx context.Context, homeDir, configDir string) error {
	svc, err := NewService(ServiceConfig{
		HomeDir:   homeDir,
		ConfigDir: configDir,
		Checker:   recoveryOnlyChecker{},
	})
	if err != nil {
		return err
	}
	return svc.Recover(ctx)
}
