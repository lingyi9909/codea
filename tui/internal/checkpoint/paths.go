package checkpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Paths struct {
	WorkspaceID  string
	Root         string
	GitDir       string
	Metadata     string
	RestoreState string
}

func canonicalRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("empty project root")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = filepath.Clean(resolved)
	}
	normalized := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		normalized = strings.ToLower(normalized)
	}
	return normalized, nil
}

func WorkspaceID(projectRoot string) (string, error) {
	canonical, err := canonicalRoot(projectRoot)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:]), nil
}

func ResolvePaths(codeaHome, projectRoot string) (Paths, error) {
	if strings.TrimSpace(codeaHome) == "" {
		return Paths{}, &Error{Code: CodeCheckpointUnavailable, Message: "CODEA_HOME is empty"}
	}
	projectAbs, err := filepath.Abs(projectRoot)
	if err != nil {
		return Paths{}, &Error{Code: CodeCheckpointUnavailable, Message: "resolve project root", Cause: err}
	}
	projectAbs = filepath.Clean(projectAbs)
	if resolved, err := filepath.EvalSymlinks(projectAbs); err == nil {
		projectAbs = filepath.Clean(resolved)
	}

	homeAbs, err := filepath.Abs(codeaHome)
	if err != nil {
		return Paths{}, &Error{Code: CodeCheckpointUnavailable, Message: "resolve CODEA_HOME", Cause: err}
	}
	homeAbs = filepath.Clean(homeAbs)
	if err := os.MkdirAll(homeAbs, 0o700); err != nil {
		return Paths{}, &Error{Code: CodeCheckpointUnavailable, Message: "create CODEA_HOME", Cause: err}
	}
	if resolved, err := filepath.EvalSymlinks(homeAbs); err == nil {
		homeAbs = filepath.Clean(resolved)
	}

	id, err := WorkspaceID(projectAbs)
	if err != nil {
		return Paths{}, &Error{Code: CodeCheckpointUnavailable, Message: "derive workspace id", Cause: err}
	}
	root := filepath.Join(homeAbs, "checkpoints", id)
	if insidePath(projectAbs, root) {
		return Paths{}, &Error{Code: CodeCheckpointUnavailable, Message: "checkpoint store must be outside the project root"}
	}
	return Paths{
		WorkspaceID:  id,
		Root:         root,
		GitDir:       filepath.Join(root, "git"),
		Metadata:     filepath.Join(root, "checkpoints.json"),
		RestoreState: filepath.Join(root, "restore-state.json"),
	}, nil
}

func insidePath(parent, child string) bool {
	p, err1 := filepath.Abs(parent)
	c, err2 := filepath.Abs(child)
	if err1 != nil || err2 != nil {
		return false
	}
	p = filepath.Clean(p)
	c = filepath.Clean(c)
	rel, err := filepath.Rel(p, c)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
