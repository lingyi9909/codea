package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RestoreResult struct {
	Target       Checkpoint
	Safety       Checkpoint
	FilesChanged int
}

type RestoreState struct {
	WorkspaceID string `json:"workspaceId"`
	TargetID    string `json:"targetId"`
	SafetyID    string `json:"safetyId"`
}

type treeEntry struct {
	Mode   string
	Type   string
	Object string
}

func (s *Service) Restore(ctx context.Context, checkpointID string) (RestoreResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !checkpointIDPattern.MatchString(checkpointID) {
		return RestoreResult{}, &Error{Code: CodeInvalidCheckpoint, Message: "checkpoint id is malformed"}
	}
	state, err := loadMetadata(s.paths.Metadata, s.paths.WorkspaceID)
	if err != nil {
		return RestoreResult{}, err
	}
	target, ok := checkpointByID(state.Checkpoints, checkpointID)
	if !ok {
		return RestoreResult{}, &Error{Code: CodeInvalidCheckpoint, Message: "checkpoint id is unknown"}
	}
	if _, err := s.commitTree(ctx, target.Commit); err != nil {
		return RestoreResult{}, &Error{Code: CodeInvalidCheckpoint, Message: "checkpoint commit is unavailable", Cause: err}
	}

	safety, err := s.createLocked(ctx, CreateRequest{
		TaskID: target.TaskID,
		TurnID: target.TurnID,
		Label:  "pre-restore " + target.ID,
		Kind:   KindSafety,
	})
	if err != nil {
		return RestoreResult{}, err
	}

	recovery := &RestoreState{WorkspaceID: s.paths.WorkspaceID, TargetID: target.ID, SafetyID: safety.ID}
	if err := writeRestoreStateAtomic(s.paths.RestoreState, *recovery); err != nil {
		return RestoreResult{}, &Error{Code: CodeRestoreInterrupted, Message: "safety checkpoint created but restore state could not be persisted", Cause: err}
	}
	s.recovery = recovery

	changed, err := s.applyRestore(ctx, target, safety)
	if err != nil {
		return RestoreResult{Target: target, Safety: safety, FilesChanged: changed}, &Error{
			Code:    CodeRestoreInterrupted,
			Message: fmt.Sprintf("restore interrupted; recover with safety checkpoint %s", safety.ID),
			Cause:   err,
		}
	}
	if _, err := s.repo.run(ctx, []string{"read-tree", target.Commit}, nil); err != nil {
		return RestoreResult{Target: target, Safety: safety, FilesChanged: changed}, &Error{
			Code:    CodeRestoreInterrupted,
			Message: fmt.Sprintf("files restored but shadow index recovery is required; safety checkpoint %s", safety.ID),
			Cause:   err,
		}
	}
	if err := os.Remove(s.paths.RestoreState); err != nil && !os.IsNotExist(err) {
		return RestoreResult{Target: target, Safety: safety, FilesChanged: changed}, &Error{
			Code:    CodeRestoreInterrupted,
			Message: fmt.Sprintf("restore completed but recovery marker cleanup failed; safety checkpoint %s", safety.ID),
			Cause:   err,
		}
	}
	s.recovery = nil
	return RestoreResult{Target: target, Safety: safety, FilesChanged: changed}, nil
}

func (s *Service) applyRestore(ctx context.Context, target, safety Checkpoint) (int, error) {
	targetEntries, err := s.treeEntries(ctx, target.Commit)
	if err != nil {
		return 0, err
	}
	safetyEntries, err := s.treeEntries(ctx, safety.Commit)
	if err != nil {
		return 0, err
	}
	protected := make(map[string]struct{}, len(safety.Skipped))
	for _, item := range safety.Skipped {
		protected[item.Path] = struct{}{}
	}

	changedSet := map[string]struct{}{}
	for path, before := range safetyEntries {
		after, ok := targetEntries[path]
		if !ok || before != after {
			changedSet[path] = struct{}{}
		}
	}
	for path, after := range targetEntries {
		before, ok := safetyEntries[path]
		if !ok || before != after {
			changedSet[path] = struct{}{}
		}
	}

	deletions := make([]string, 0)
	writes := make([]string, 0)
	for path := range changedSet {
		if _, skip := protected[path]; skip {
			continue
		}
		if _, exists := targetEntries[path]; exists {
			writes = append(writes, path)
		} else {
			deletions = append(deletions, path)
		}
	}
	sort.Slice(deletions, func(i, j int) bool {
		di := strings.Count(deletions[i], "/")
		dj := strings.Count(deletions[j], "/")
		if di == dj {
			return deletions[i] > deletions[j]
		}
		return di > dj
	})
	sort.Strings(writes)

	applied := 0
	for _, rel := range deletions {
		full, err := s.safeProjectPath(rel)
		if err != nil {
			return applied, err
		}
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return applied, err
		}
		removeEmptyParents(filepath.Dir(full), s.repo.ProjectRoot())
		applied++
	}
	for _, rel := range writes {
		entry := targetEntries[rel]
		if entry.Type != "blob" || !commitPattern.MatchString(entry.Object) {
			return applied, &Error{Code: CodeStateCorrupt, Message: "target checkpoint tree contains unsupported entry"}
		}
		result, err := s.repo.run(ctx, []string{"cat-file", "blob", entry.Object}, nil)
		if err != nil {
			return applied, err
		}
		if int64(len(result.Stdout)) > s.maxFileSize {
			return applied, &Error{Code: CodeStateCorrupt, Message: "target checkpoint blob exceeds checkpoint size policy"}
		}
		full, err := s.safeProjectPath(rel)
		if err != nil {
			return applied, err
		}
		mode := os.FileMode(0o644)
		if entry.Mode == "100755" {
			mode = 0o755
		}
		if err := writeFileAtomic(full, result.Stdout, mode); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}

func (s *Service) treeEntries(ctx context.Context, commit string) (map[string]treeEntry, error) {
	if !commitPattern.MatchString(commit) {
		return nil, &Error{Code: CodeStateCorrupt, Message: "invalid checkpoint commit id"}
	}
	result, err := s.repo.run(ctx, []string{"ls-tree", "-r", "-z", commit}, nil)
	if err != nil {
		return nil, &Error{Code: CodeStateCorrupt, Message: "read checkpoint tree", Cause: err}
	}
	entries := map[string]treeEntry{}
	for _, record := range strings.Split(string(result.Stdout), "\x00") {
		if record == "" {
			continue
		}
		header, path, ok := strings.Cut(record, "\t")
		if !ok {
			return nil, &Error{Code: CodeStateCorrupt, Message: "checkpoint tree record is malformed"}
		}
		fields := strings.Fields(header)
		if len(fields) != 3 {
			return nil, &Error{Code: CodeStateCorrupt, Message: "checkpoint tree header is malformed"}
		}
		if _, err := s.safeProjectPath(path); err != nil {
			return nil, err
		}
		entries[path] = treeEntry{Mode: fields[0], Type: fields[1], Object: fields[2]}
	}
	return entries, nil
}

func (s *Service) safeProjectPath(rel string) (string, error) {
	if rel == "" || filepath.IsAbs(filepath.FromSlash(rel)) {
		return "", &Error{Code: CodeStateCorrupt, Message: "checkpoint path is invalid"}
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", &Error{Code: CodeStateCorrupt, Message: "checkpoint path escapes project root"}
	}
	full := filepath.Join(s.repo.ProjectRoot(), clean)
	if !insidePath(s.repo.ProjectRoot(), full) {
		return "", &Error{Code: CodeStateCorrupt, Message: "checkpoint path escapes project root"}
	}
	return full, nil
}

func checkpointByID(items []Checkpoint, id string) (Checkpoint, bool) {
	for _, cp := range items {
		if cp.ID == id {
			return cp, true
		}
	}
	return Checkpoint{}, false
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	f, err := os.CreateTemp(dir, ".codea-restore-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(tmp) }
	if err := f.Chmod(mode); err != nil {
		cleanup()
		return err
	}
	if _, err := f.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			_ = os.Remove(tmp)
			return err
		}
		if renameErr := os.Rename(tmp, path); renameErr != nil {
			_ = os.Remove(tmp)
			return renameErr
		}
	}
	return os.Chmod(path, mode)
}

func removeEmptyParents(dir, stop string) {
	stop = filepath.Clean(stop)
	for {
		dir = filepath.Clean(dir)
		if dir == stop || !insidePath(stop, dir) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func writeRestoreStateAtomic(path string, state RestoreState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".restore-state-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(tmp) }
	if err := f.Chmod(0o600); err != nil {
		cleanup()
		return err
	}
	if _, err := f.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func loadRestoreState(path, workspaceID string) (*RestoreState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{Code: CodeStateCorrupt, Message: "read restore state", Cause: err}
	}
	var state RestoreState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, &Error{Code: CodeStateCorrupt, Message: "restore state is invalid JSON", Cause: err}
	}
	if state.WorkspaceID != workspaceID || !checkpointIDPattern.MatchString(state.TargetID) || !checkpointIDPattern.MatchString(state.SafetyID) {
		return nil, &Error{Code: CodeStateCorrupt, Message: "restore state is invalid"}
	}
	return &state, nil
}

func (s *Service) Recovery() *RestoreState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recovery == nil {
		return nil
	}
	copy := *s.recovery
	return &copy
}

func (s *Service) RecoveryGuidance() string {
	recovery := s.Recovery()
	if recovery == nil {
		return ""
	}
	return fmt.Sprintf("Previous restore was interrupted. Safety checkpoint %s can recover the pre-restore source state.", recovery.SafetyID)
}
