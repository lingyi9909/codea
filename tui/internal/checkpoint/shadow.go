package checkpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var shadowExcludePatterns = []string{
	".git/",
	"target/",
	"build/",
	"dist/",
	"node_modules/",
	".codea/",
	".env",
	".env.*",
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",
	"credentials*",
	"secrets*",
}

type ShadowRepo struct {
	gitDir      string
	projectRoot string
	runner      Runner
	paths       Paths
}

func OpenOrInit(ctx context.Context, codeaHome, projectRoot string, runner Runner) (*ShadowRepo, error) {
	if runner == nil {
		runner = NewGitRunner()
	}
	paths, err := ResolvePaths(codeaHome, projectRoot)
	if err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, &Error{Code: CodeCheckpointUnavailable, Message: "resolve project root", Cause: err}
	}
	absRoot = filepath.Clean(absRoot)
	if resolved, statErr := filepath.EvalSymlinks(absRoot); statErr == nil {
		absRoot = filepath.Clean(resolved)
	}
	if info, statErr := os.Stat(absRoot); statErr != nil || !info.IsDir() {
		if statErr == nil {
			statErr = fmt.Errorf("project root is not a directory")
		}
		return nil, &Error{Code: CodeCheckpointUnavailable, Message: "project root is unavailable", Cause: statErr}
	}
	if err := os.MkdirAll(paths.Root, 0o700); err != nil {
		return nil, &Error{Code: CodeCheckpointUnavailable, Message: "create checkpoint workspace", Cause: err}
	}
	repo := &ShadowRepo{gitDir: paths.GitDir, projectRoot: absRoot, runner: runner, paths: paths}
	if _, statErr := os.Stat(filepath.Join(paths.GitDir, "HEAD")); statErr != nil {
		if !os.IsNotExist(statErr) {
			return nil, &Error{Code: CodeCheckpointUnavailable, Message: "inspect shadow git repository", Cause: statErr}
		}
		if _, runErr := runner.Run(ctx, []string{"init", "--bare", paths.GitDir}, nil); runErr != nil {
			return nil, wrapUnavailable(runErr, "initialize shadow git repository")
		}
	}
	for _, cfg := range [][]string{
		{"config", "user.name", "Codea Checkpoint"},
		{"config", "user.email", "checkpoint@codea.local"},
		{"config", "commit.gpgSign", "false"},
	} {
		if _, runErr := runner.Run(ctx, append([]string{"--git-dir=" + paths.GitDir}, cfg...), nil); runErr != nil {
			return nil, wrapUnavailable(runErr, "configure shadow git repository")
		}
	}
	if err := writeShadowExcludes(paths.GitDir); err != nil {
		return nil, &Error{Code: CodeCheckpointUnavailable, Message: "write shadow excludes", Cause: err}
	}
	return repo, nil
}

func writeShadowExcludes(gitDir string) error {
	infoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return err
	}
	data := strings.Join(shadowExcludePatterns, "\n") + "\n"
	return os.WriteFile(filepath.Join(infoDir, "exclude"), []byte(data), 0o600)
}

func wrapUnavailable(err error, action string) error {
	if IsCode(err, CodeCheckpointUnavailable) {
		return err
	}
	return &Error{Code: CodeCheckpointUnavailable, Message: action, Cause: err}
}

func (s *ShadowRepo) GitDir() string {
	if s == nil {
		return ""
	}
	return s.gitDir
}
func (s *ShadowRepo) ProjectRoot() string {
	if s == nil {
		return ""
	}
	return s.projectRoot
}
func (s *ShadowRepo) Paths() Paths {
	if s == nil {
		return Paths{}
	}
	return s.paths
}

func (s *ShadowRepo) args(args ...string) []string {
	base := []string{"-C", s.projectRoot, "--literal-pathspecs", "--git-dir=" + s.gitDir, "--work-tree=" + s.projectRoot}
	return append(base, args...)
}

func (s *ShadowRepo) run(ctx context.Context, args []string, stdin []byte) (Result, error) {
	return s.runner.Run(ctx, s.args(args...), stdin)
}
