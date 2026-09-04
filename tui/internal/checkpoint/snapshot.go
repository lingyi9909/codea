package checkpoint

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMaxFileSize     int64 = 5 * 1024 * 1024
	DefaultBinaryThreshold int64 = 1024 * 1024
	binaryProbeBytes             = 8192
	DefaultRetention             = 20
)

type CreateRequest struct {
	TaskID string
	TurnID string
	Label  string
	Kind   Kind
}

type Service struct {
	repo            *ShadowRepo
	paths           Paths
	now             func() time.Time
	maxFileSize     int64
	binaryThreshold int64
	retention       int
	mu              sync.Mutex
	recovery        *RestoreState
}

func NewService(ctx context.Context, codeaHome, projectRoot string, runner Runner) (*Service, error) {
	repo, err := OpenOrInit(ctx, codeaHome, projectRoot, runner)
	if err != nil {
		return nil, err
	}
	s := &Service{
		repo:            repo,
		paths:           repo.Paths(),
		now:             time.Now,
		maxFileSize:     DefaultMaxFileSize,
		binaryThreshold: DefaultBinaryThreshold,
		retention:       DefaultRetention,
	}
	if recovery, err := loadRestoreState(s.paths.RestoreState, s.paths.WorkspaceID); err != nil {
		return nil, err
	} else {
		s.recovery = recovery
	}
	return s, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createLocked(ctx, req)
}

func (s *Service) createLocked(ctx context.Context, req CreateRequest) (Checkpoint, error) {
	if !validKind(req.Kind) {
		return Checkpoint{}, &Error{Code: CodeInvalidCheckpoint, Message: "invalid checkpoint kind"}
	}
	state, err := loadMetadata(s.paths.Metadata, s.paths.WorkspaceID)
	if err != nil {
		return Checkpoint{}, err
	}
	latest := latestCheckpoint(state.Checkpoints)
	if err := s.resetShadowIndex(ctx, latest); err != nil {
		return Checkpoint{}, err
	}

	candidates, skipped, err := s.enumerateProjectFiles()
	if err != nil {
		return Checkpoint{}, err
	}
	if err := s.stageSnapshot(ctx, candidates, skipped); err != nil {
		return Checkpoint{}, err
	}

	treeResult, err := s.repo.run(ctx, []string{"write-tree"}, nil)
	if err != nil {
		return Checkpoint{}, wrapUnavailable(err, "write shadow checkpoint tree")
	}
	tree := strings.TrimSpace(string(treeResult.Stdout))
	if !commitPattern.MatchString(tree) {
		return Checkpoint{}, &Error{Code: CodeStateCorrupt, Message: "shadow tree id is invalid"}
	}

	id := nextCheckpointID(&state)
	commit := ""
	if latest != nil {
		latestTree, treeErr := s.commitTree(ctx, latest.Commit)
		if treeErr != nil {
			return Checkpoint{}, treeErr
		}
		if latestTree == tree {
			commit = latest.Commit
		}
	}
	if commit == "" {
		args := []string{"commit-tree", tree}
		if latest != nil {
			args = append(args, "-p", latest.Commit)
		}
		msg := fmt.Sprintf("codea checkpoint %s %s", id, req.Kind)
		if token := safeToken(req.TaskID); token != "" {
			msg += " task=" + token
		}
		result, commitErr := s.repo.run(ctx, args, []byte(msg+"\n"))
		if commitErr != nil {
			return Checkpoint{}, wrapUnavailable(commitErr, "create shadow checkpoint commit")
		}
		commit = strings.TrimSpace(string(result.Stdout))
		if !commitPattern.MatchString(commit) {
			return Checkpoint{}, &Error{Code: CodeStateCorrupt, Message: "shadow commit id is invalid"}
		}
	}

	cp := Checkpoint{
		ID: id, TaskID: req.TaskID, TurnID: req.TurnID, Commit: commit,
		Label: req.Label, Kind: req.Kind, CreatedAt: s.now().UTC(), Skipped: skipped,
	}
	state.Checkpoints = append(state.Checkpoints, cp)
	state.Checkpoints = pruneCheckpointRecords(state.Checkpoints, s.retention, s.recovery)
	if err := writeMetadataAtomic(s.paths.Metadata, state); err != nil {
		return Checkpoint{}, err
	}
	return cp, nil
}

func (s *Service) List(ctx context.Context) ([]Checkpoint, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := loadMetadata(s.paths.Metadata, s.paths.WorkspaceID)
	if err != nil {
		return nil, err
	}
	out := append([]Checkpoint(nil), state.Checkpoints...)
	return out, nil
}

func latestCheckpoint(items []Checkpoint) *Checkpoint {
	if len(items) == 0 {
		return nil
	}
	cp := items[len(items)-1]
	return &cp
}

func (s *Service) resetShadowIndex(ctx context.Context, latest *Checkpoint) error {
	args := []string{"read-tree", "--empty"}
	if latest != nil {
		args = []string{"read-tree", latest.Commit}
	}
	if _, err := s.repo.run(ctx, args, nil); err != nil {
		return wrapUnavailable(err, "reset shadow checkpoint index")
	}
	return nil
}

func (s *Service) stageSnapshot(ctx context.Context, candidates []string, skipped []SkippedPath) error {
	if len(skipped) > 0 {
		var remove bytes.Buffer
		for _, item := range skipped {
			remove.WriteString(item.Path)
			remove.WriteByte(0)
		}
		if _, err := s.repo.run(ctx, []string{"update-index", "--force-remove", "-z", "--stdin"}, remove.Bytes()); err != nil {
			return wrapUnavailable(err, "remove skipped paths from shadow index")
		}
	}
	if _, err := s.repo.run(ctx, []string{"add", "-u"}, nil); err != nil {
		return wrapUnavailable(err, "stage shadow deletions and modifications")
	}
	if len(candidates) == 0 {
		return nil
	}
	var pathspec bytes.Buffer
	for _, rel := range candidates {
		pathspec.WriteString(rel)
		pathspec.WriteByte(0)
	}
	if _, err := s.repo.run(ctx, []string{"add", "--pathspec-from-file=-", "--pathspec-file-nul"}, pathspec.Bytes()); err != nil {
		return wrapUnavailable(err, "stage shadow checkpoint files")
	}
	return nil
}

func (s *Service) enumerateProjectFiles() ([]string, []SkippedPath, error) {
	var candidates []string
	var skipped []SkippedPath
	err := filepath.WalkDir(s.repo.ProjectRoot(), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == s.repo.ProjectRoot() {
			return nil
		}
		rel, err := filepath.Rel(s.repo.ProjectRoot(), path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if excludeReason(rel, true) != "" {
				return filepath.SkipDir
			}
			return nil
		}
		reason := excludeReason(rel, false)
		if reason != "" {
			skipped = append(skipped, SkippedPath{Path: rel, Reason: reason})
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			skipped = append(skipped, SkippedPath{Path: rel, Reason: "symlink"})
			return nil
		}
		if !info.Mode().IsRegular() {
			skipped = append(skipped, SkippedPath{Path: rel, Reason: "non-regular"})
			return nil
		}
		if info.Size() > s.maxFileSize {
			skipped = append(skipped, SkippedPath{Path: rel, Reason: "large-file"})
			return nil
		}
		if info.Size() > s.binaryThreshold {
			binary, err := fileLooksBinary(path)
			if err != nil {
				return err
			}
			if binary {
				skipped = append(skipped, SkippedPath{Path: rel, Reason: "binary-file"})
				return nil
			}
		}
		candidates = append(candidates, rel)
		return nil
	})
	if err != nil {
		return nil, nil, &Error{Code: CodeCheckpointUnavailable, Message: "enumerate project files", Cause: err}
	}
	sort.Strings(candidates)
	sort.Slice(skipped, func(i, j int) bool { return skipped[i].Path < skipped[j].Path })
	return candidates, skipped, nil
}

func excludeReason(rel string, isDir bool) string {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		switch part {
		case ".git", "target", "build", "dist", "node_modules", ".codea":
			return "excluded"
		}
	}
	if isDir {
		return ""
	}
	base := filepath.Base(filepath.FromSlash(rel))
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return "sensitive"
	}
	lower := strings.ToLower(base)
	if strings.HasSuffix(lower, ".pem") || strings.HasSuffix(lower, ".key") || strings.HasSuffix(lower, ".p12") || strings.HasSuffix(lower, ".pfx") || strings.HasPrefix(lower, "credentials") || strings.HasPrefix(lower, "secrets") {
		return "sensitive"
	}
	return ""
}

func fileLooksBinary(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, binaryProbeBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false, err
	}
	return bytes.IndexByte(buf[:n], 0) >= 0, nil
}

func (s *Service) commitTree(ctx context.Context, commit string) (string, error) {
	if !commitPattern.MatchString(commit) {
		return "", &Error{Code: CodeStateCorrupt, Message: "invalid checkpoint commit id"}
	}
	result, err := s.repo.run(ctx, []string{"cat-file", "-p", commit}, nil)
	if err != nil {
		return "", &Error{Code: CodeStateCorrupt, Message: "checkpoint commit is missing", Cause: err}
	}
	first, _, _ := strings.Cut(string(result.Stdout), "\n")
	if !strings.HasPrefix(first, "tree ") {
		return "", &Error{Code: CodeStateCorrupt, Message: "checkpoint commit has no tree"}
	}
	tree := strings.TrimSpace(strings.TrimPrefix(first, "tree "))
	if !commitPattern.MatchString(tree) {
		return "", &Error{Code: CodeStateCorrupt, Message: "checkpoint tree id is invalid"}
	}
	return tree, nil
}

func safeToken(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	return b.String()
}
