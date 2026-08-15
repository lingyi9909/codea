package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Store persists per-skill enable/disable overrides. A skill absent from the
// store uses its source default.
type Store interface {
	Load() (map[string]bool, error)
	Save(overrides map[string]bool) error
}

// FileStore persists overrides as a JSON object mapping skill name -> enabled.
type FileStore struct {
	path string
}

// NewFileStore returns a FileStore backed by path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

func (s *FileStore) Load() (map[string]bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}
	overrides := map[string]bool{}
	if err := json.Unmarshal(data, &overrides); err != nil {
		return nil, err
	}
	return overrides, nil
}

func (s *FileStore) Save(overrides map[string]bool) error {
	data, err := json.MarshalIndent(overrides, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o644)
}
