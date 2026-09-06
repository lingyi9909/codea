package modelprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"codea/tui/internal/runtime"
)

const CodeModelProfileCorrupt = "MODEL_PROFILE_CORRUPT"

// Store persists safe per-model qualification metadata under CODEA_HOME.
type Store struct {
	home string
}

func NewStore(codeaHome string) *Store { return &Store{home: codeaHome} }

// ProfilePath derives a path-safe identifier from provider+NUL+model.
func ProfilePath(codeaHome string, ref runtime.ModelRef) string {
	sum := sha256.Sum256([]byte(ref.ProviderID + "\x00" + ref.ModelID))
	return filepath.Join(codeaHome, "model-profiles", hex.EncodeToString(sum[:])+".json")
}

func (s *Store) Load(ref runtime.ModelRef) (ModelCapabilityProfile, bool, error) {
	path := ProfilePath(s.home, ref)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ModelCapabilityProfile{}, false, nil
	}
	if err != nil {
		return ModelCapabilityProfile{}, false, corruptError("read model profile", err)
	}

	var profile ModelCapabilityProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return ModelCapabilityProfile{}, false, corruptError("model profile is invalid JSON", err)
	}
	if profile.Version != ProfileVersion {
		return ModelCapabilityProfile{}, false, nil
	}
	if profile.Model != ref || !completeAndConsistent(profile) {
		return ModelCapabilityProfile{}, false, corruptError("model profile is inconsistent", nil)
	}
	return profile, true, nil
}

func (s *Store) Save(profile ModelCapabilityProfile) error {
	if profile.Version != ProfileVersion || profile.Model.ProviderID == "" || profile.Model.ModelID == "" || profile.QualifiedAt.IsZero() || !completeAndConsistent(profile) {
		return corruptError("refuse incomplete or inconsistent model profile", nil)
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return corruptError("encode model profile", err)
	}
	data = append(data, '\n')

	path := ProfilePath(s.home, profile.Model)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return corruptError("create model profile directory", err)
	}
	f, err := os.CreateTemp(dir, ".model-profile-*.tmp")
	if err != nil {
		return corruptError("create model profile temp file", err)
	}
	tmp := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}
	if err := f.Chmod(0o600); err != nil {
		cleanup()
		return corruptError("secure model profile temp file", err)
	}
	if _, err := f.Write(data); err != nil {
		cleanup()
		return corruptError("write model profile", err)
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return corruptError("sync model profile", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return corruptError("close model profile", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return corruptError("replace model profile", err)
	}
	return nil
}

func completeAndConsistent(profile ModelCapabilityProfile) bool {
	levels := []CapabilityLevel{profile.ToolCalling, profile.StructuredOutput, profile.PatchFollowing, profile.Planning}
	for _, level := range levels {
		if level == CapabilityUnknown || !validLevel(level) {
			return false
		}
	}
	return profile.Overall == Aggregate(levels...) && profile.Overall != CapabilityUnknown
}

func corruptError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%s: %s", CodeModelProfileCorrupt, message)
	}
	return fmt.Errorf("%s: %s: %w", CodeModelProfileCorrupt, message, cause)
}
