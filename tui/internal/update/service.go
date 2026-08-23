package update

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type CheckPhase string

const (
	CheckPreSwitch  CheckPhase = "pre-switch"
	CheckPostSwitch CheckPhase = "post-switch"
)

type Candidate struct{ Version, VersionDir, ConfigDir string }
type CandidateChecker interface {
	Check(context.Context, CheckPhase, Candidate) error
}
type BasicChecker struct{}

func (BasicChecker) Check(_ context.Context, _ CheckPhase, c Candidate) error {
	if c.Version == "" || c.VersionDir == "" || c.ConfigDir == "" {
		return fmt.Errorf("candidate is incomplete")
	}
	for _, p := range []string{"VERSION", filepath.Join("plugins", "index.js"), "agents", "skills"} {
		if _, err := os.Stat(filepath.Join(c.VersionDir, p)); err != nil {
			return fmt.Errorf("candidate missing %s: %w", p, err)
		}
	}
	codeaName, runtimeName := "codea", "opencode"
	if runtime.GOOS == "windows" {
		codeaName += ".exe"
		runtimeName += ".exe"
	}
	for _, p := range []string{filepath.Join("bin", codeaName), filepath.Join("bin", runtimeName)} {
		if st, err := os.Stat(filepath.Join(c.VersionDir, p)); err != nil || st.IsDir() {
			return fmt.Errorf("candidate missing executable %s", p)
		}
	}
	b, err := os.ReadFile(filepath.Join(c.VersionDir, "VERSION"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(b)) != c.Version {
		return fmt.Errorf("candidate VERSION mismatch")
	}
	if st, err := os.Stat(c.ConfigDir); err != nil || !st.IsDir() {
		return fmt.Errorf("candidate config dir unavailable")
	}
	return nil
}

type ServiceConfig struct {
	HomeDir    string
	ConfigDir  string
	Checker    CandidateChecker
	Migrations *MigrationRegistry
	Switcher   Switcher
	Now        func() time.Time
}
type UpdateService struct {
	home, configDir string
	checker         CandidateChecker
	migrations      *MigrationRegistry
	switcher        Switcher
	versions        *VersionManager
	journal         *Journal
	verifier        Verifier
	now             func() time.Time
}

func NewService(cfg ServiceConfig) (*UpdateService, error) {
	if strings.TrimSpace(cfg.HomeDir) == "" {
		return nil, fmt.Errorf("home dir is required")
	}
	home, err := filepath.Abs(cfg.HomeDir)
	if err != nil {
		return nil, err
	}
	configDir := cfg.ConfigDir
	if configDir == "" {
		configDir = filepath.Join(home, "runtime-config")
	}
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(home, configDir)
	}
	if cfg.Checker == nil {
		return nil, fmt.Errorf("candidate checker is required")
	}
	if cfg.Migrations == nil {
		cfg.Migrations = NewMigrationRegistry()
	}
	if cfg.Switcher == nil {
		cfg.Switcher = NewPlatformSwitcher(home)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &UpdateService{home: home, configDir: configDir, checker: cfg.Checker, migrations: cfg.Migrations, switcher: cfg.Switcher, versions: NewVersionManager(home), journal: NewJournal(home), now: cfg.Now}, nil
}

func (s *UpdateService) Upgrade(ctx context.Context, packagePath string) (err error) {
	lock, err := acquireUpdateLock(s.home)
	if err != nil {
		return err
	}
	defer lock.Release()
	if pending, err := s.journal.RecoverPending(); err != nil {
		return err
	} else if pending != nil {
		if err := s.rollbackTx(ctx, pending, true); err != nil {
			return fmt.Errorf("recover pending transaction: %w", err)
		}
	}
	currentPath, err := s.switcher.Current()
	if err != nil {
		return fmt.Errorf("resolve current version: %w", err)
	}
	fromVersion := filepath.Base(filepath.Clean(currentPath))
	txID, err := newTxID(s.now())
	if err != nil {
		return err
	}
	stage := filepath.Join(s.home, "staging", txID)
	defer os.RemoveAll(stage)
	releaseRoot, err := PreparePackage(packagePath, stage)
	if err != nil {
		return fmt.Errorf("prepare package: %w", err)
	}
	info, err := s.verifier.Verify(releaseRoot)
	if err != nil {
		return fmt.Errorf("verify package: %w", err)
	}
	if info.Version == fromVersion {
		return fmt.Errorf("target version %s is already current", info.Version)
	}
	if cmp, cmpErr := compareVersions(info.Version, fromVersion); cmpErr != nil {
		return cmpErr
	} else if cmp <= 0 {
		return fmt.Errorf("upgrade target %s must be newer than current %s", info.Version, fromVersion)
	}
	if s.versions.Exists(info.Version) {
		return fmt.Errorf("target version already installed: %s", info.Version)
	}
	tempConfig := filepath.Join(s.home, "transactions", txID, "config-c2-temp")
	if err := cloneConfigDir(s.configDir, tempConfig); err != nil {
		return fmt.Errorf("clone config: %w", err)
	}
	if err := s.migrateConfig(tempConfig, info.ConfigSchemaVersion); err != nil {
		return fmt.Errorf("migrate config: %w", err)
	}
	candidate := Candidate{Version: info.Version, VersionDir: releaseRoot, ConfigDir: tempConfig}
	if err := s.checker.Check(ctx, CheckPreSwitch, candidate); err != nil {
		return fmt.Errorf("pre-switch doctor: %w", err)
	}
	backup := filepath.Join(s.home, "backups", txID, "runtime-config")
	tx := &Transaction{ID: txID, FromVersion: fromVersion, ToVersion: info.Version, ConfigBackupPath: backup, StartedAt: s.now().UTC()}
	if err := s.journal.Begin(tx); err != nil {
		return err
	}
	rollbackOnFailure := true
	defer func() {
		if err != nil && rollbackOnFailure {
			rbErr := s.rollbackTx(context.Background(), tx, true)
			if rbErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
			}
		}
	}()
	tx.MarkStep("install-version", TxPending, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	target, err := s.versions.Install(releaseRoot, info.Version)
	if err != nil {
		return err
	}
	tx.MarkStep("install-version", TxCommitted, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	tx.MarkStep("switch-current", TxPending, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	if err := s.switcher.Switch(target); err != nil {
		return err
	}
	tx.MarkStep("switch-current", TxCommitted, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	candidate.VersionDir = target
	if err := s.checker.Check(ctx, CheckPostSwitch, candidate); err != nil {
		return fmt.Errorf("post-switch doctor: %w", err)
	}
	tx.MarkStep("post-doctor", TxCommitted, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	tx.MarkStep("commit-config", TxPending, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	if err := commitConfig(s.configDir, tempConfig, backup); err != nil {
		return err
	}
	tx.MarkStep("commit-config", TxCommitted, "")
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	tx.Status = TxCommitted
	if err := s.journal.Save(tx); err != nil {
		return err
	}
	rollbackOnFailure = false
	return nil
}

func (s *UpdateService) Recover(ctx context.Context) error {
	lock, err := acquireUpdateLock(s.home)
	if err != nil {
		return err
	}
	defer lock.Release()
	tx, err := s.journal.RecoverPending()
	if err != nil {
		return err
	}
	if tx == nil {
		return nil
	}
	return s.rollbackTx(ctx, tx, true)
}
func (s *UpdateService) Rollback(ctx context.Context) error {
	lock, err := acquireUpdateLock(s.home)
	if err != nil {
		return err
	}
	defer lock.Release()
	tx, err := s.journal.Load()
	if err != nil {
		return err
	}
	if tx.Status != TxCommitted {
		return fmt.Errorf("last transaction is not committed")
	}
	if !s.versions.Exists(tx.FromVersion) {
		return fmt.Errorf("rollback version missing: %s", tx.FromVersion)
	}
	if tx.ConfigBackupPath == "" {
		return fmt.Errorf("rollback config backup missing from journal")
	}
	if _, err := os.Stat(tx.ConfigBackupPath); err != nil {
		return fmt.Errorf("rollback config backup unavailable: %w", err)
	}
	return s.rollbackTx(ctx, tx, false)
}

func (s *UpdateService) migrateConfig(configDir string, target int) error {
	p := filepath.Join(configDir, "codea", "config.json")
	data, err := os.ReadFile(p)
	exists := true
	if os.IsNotExist(err) {
		exists = false
		data = []byte(`{"schemaVersion":1}`)
	} else if err != nil {
		return err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("decode codea config: %w", err)
	}
	from := 1
	if raw, ok := cfg["schemaVersion"]; ok {
		switch v := raw.(type) {
		case float64:
			from = int(v)
		case int:
			from = v
		case string:
			n, e := strconv.Atoi(v)
			if e != nil {
				return fmt.Errorf("invalid schemaVersion")
			}
			from = n
		default:
			return fmt.Errorf("invalid schemaVersion type")
		}
	}
	if target <= 0 {
		target = from
	}
	out, err := s.migrations.Migrate(cfg, from, target)
	if err != nil {
		return err
	}
	if from == target && !exists {
		return nil
	}
	out["schemaVersion"] = target
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return writeFileAtomic(p, b, 0o600)
}

func cloneConfigDir(src, dst string) error {
	_ = os.RemoveAll(dst)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return os.MkdirAll(dst, 0o700)
	} else if err != nil {
		return err
	}
	return copyTree(src, dst)
}
func commitConfig(current, temp, backup string) error {
	if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
		return err
	}
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(current); err == nil {
		if err := os.Rename(current, backup); err != nil {
			return fmt.Errorf("backup current config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(temp, current); err != nil {
		_ = os.Rename(backup, current)
		return fmt.Errorf("activate migrated config: %w", err)
	}
	return nil
}
func restoreConfig(current, backup string) error {
	if backup == "" {
		return nil
	}
	if _, err := os.Stat(backup); err != nil {
		return err
	}
	tmp := current + ".rollback-old"
	_ = os.RemoveAll(tmp)
	if _, err := os.Stat(current); err == nil {
		if err := os.Rename(current, tmp); err != nil {
			return err
		}
	}
	if err := cloneConfigDir(backup, current); err != nil {
		_ = os.Rename(tmp, current)
		return err
	}
	_ = os.RemoveAll(tmp)
	return nil
}
func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return replaceFileAtomic(name, path)
}
func compareVersions(a, b string) (int, error) {
	parse := func(v string) ([3]int, error) {
		var out [3]int
		core := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(v), "v"), "-", 2)[0]
		parts := strings.Split(core, ".")
		if len(parts) != 3 {
			return out, fmt.Errorf("version %q is not major.minor.patch", v)
		}
		for i, part := range parts {
			n, err := strconv.Atoi(part)
			if err != nil || n < 0 {
				return out, fmt.Errorf("invalid version %q", v)
			}
			out[i] = n
		}
		return out, nil
	}
	av, err := parse(a)
	if err != nil {
		return 0, err
	}
	bv, err := parse(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1, nil
		}
		if av[i] > bv[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func newTxID(now time.Time) (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return now.UTC().Format("20060102T150405.000000000") + "-" + hex.EncodeToString(b), nil
}

func (s *UpdateService) rollbackTx(_ context.Context, tx *Transaction, failedUpgrade bool) error {
	if tx == nil {
		return nil
	}
	var errs []error

	if status, ok := tx.StepStatus("commit-config"); ok && (status == TxCommitted || status == TxPending) {
		if _, err := os.Stat(tx.ConfigBackupPath); err == nil {
			if err := restoreConfig(s.configDir, tx.ConfigBackupPath); err != nil {
				errs = append(errs, fmt.Errorf("restore config: %w", err))
			}
		} else if !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("inspect config backup: %w", err))
		}
	}

	if status, ok := tx.StepStatus("switch-current"); ok && (status == TxCommitted || status == TxPending) && tx.FromVersion != "" {
		shouldSwitch := status == TxCommitted
		if !shouldSwitch {
			if current, err := s.switcher.Current(); err == nil && filepath.Base(current) == tx.ToVersion {
				shouldSwitch = true
			}
		}
		if shouldSwitch {
			old := s.versions.Path(tx.FromVersion)
			if _, err := os.Stat(old); err != nil {
				errs = append(errs, fmt.Errorf("previous version unavailable: %w", err))
			} else if err := s.switcher.Switch(old); err != nil {
				errs = append(errs, fmt.Errorf("switch previous version: %w", err))
			}
		}
	}

	if failedUpgrade {
		if status, ok := tx.StepStatus("install-version"); ok && (status == TxCommitted || status == TxPending) && tx.ToVersion != "" {
			if s.versions.Exists(tx.ToVersion) {
				if err := s.versions.Remove(tx.ToVersion); err != nil {
					errs = append(errs, fmt.Errorf("remove failed target: %w", err))
				}
			}
		}
	}

	if len(errs) > 0 {
		tx.MarkStep("rollback", TxPending, errors.Join(errs...).Error())
		_ = s.journal.Save(tx)
		return errors.Join(errs...)
	}
	tx.Status = TxRolledBack
	tx.MarkStep("rollback", TxRolledBack, "")
	return s.journal.Save(tx)
}
