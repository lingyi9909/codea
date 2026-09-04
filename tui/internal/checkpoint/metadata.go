package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const metadataVersion = 1

var (
	checkpointIDPattern = regexp.MustCompile(`^cp-[0-9]{6}$`)
	commitPattern       = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
)

type metadataState struct {
	Version      int          `json:"version"`
	WorkspaceID  string       `json:"workspaceId"`
	NextSequence int          `json:"nextSequence"`
	Checkpoints  []Checkpoint `json:"checkpoints"`
}

func newMetadataState(workspaceID string) metadataState {
	return metadataState{Version: metadataVersion, WorkspaceID: workspaceID, NextSequence: 1, Checkpoints: []Checkpoint{}}
}

func loadMetadata(path, workspaceID string) (metadataState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return newMetadataState(workspaceID), nil
	}
	if err != nil {
		return metadataState{}, &Error{Code: CodeStateCorrupt, Message: "read checkpoint metadata", Cause: err}
	}
	var state metadataState
	if err := json.Unmarshal(data, &state); err != nil {
		return metadataState{}, &Error{Code: CodeStateCorrupt, Message: "checkpoint metadata is invalid JSON", Cause: err}
	}
	if state.Version != metadataVersion || state.WorkspaceID != workspaceID || state.NextSequence < 1 {
		return metadataState{}, &Error{Code: CodeStateCorrupt, Message: "checkpoint metadata header is invalid"}
	}
	for _, cp := range state.Checkpoints {
		if !checkpointIDPattern.MatchString(cp.ID) || !commitPattern.MatchString(cp.Commit) || !validKind(cp.Kind) {
			return metadataState{}, &Error{Code: CodeStateCorrupt, Message: "checkpoint metadata record is invalid"}
		}
	}
	return state, nil
}

func writeMetadataAtomic(path string, state metadataState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return &Error{Code: CodeStateCorrupt, Message: "encode checkpoint metadata", Cause: err}
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return &Error{Code: CodeStateCorrupt, Message: "create checkpoint metadata directory", Cause: err}
	}
	f, err := os.CreateTemp(dir, ".checkpoints-*.tmp")
	if err != nil {
		return &Error{Code: CodeStateCorrupt, Message: "create checkpoint metadata temp file", Cause: err}
	}
	tmp := f.Name()
	cleanup := func() { _ = f.Close(); _ = os.Remove(tmp) }
	if err := f.Chmod(0o600); err != nil {
		cleanup()
		return &Error{Code: CodeStateCorrupt, Message: "secure checkpoint metadata temp file", Cause: err}
	}
	if _, err := f.Write(data); err != nil {
		cleanup()
		return &Error{Code: CodeStateCorrupt, Message: "write checkpoint metadata", Cause: err}
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return &Error{Code: CodeStateCorrupt, Message: "sync checkpoint metadata", Cause: err}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return &Error{Code: CodeStateCorrupt, Message: "close checkpoint metadata", Cause: err}
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return &Error{Code: CodeStateCorrupt, Message: "replace checkpoint metadata", Cause: err}
	}
	return nil
}

func validKind(kind Kind) bool {
	switch kind {
	case KindBaseline, KindManual, KindFinal, KindSafety:
		return true
	default:
		return false
	}
}

func nextCheckpointID(state *metadataState) string {
	id := fmt.Sprintf("cp-%06d", state.NextSequence)
	state.NextSequence++
	return id
}
