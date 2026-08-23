package opencode

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// MergePluginConfig preserves every existing OpenCode configuration field and
// replaces only the Codea-managed plugin field. Existing invalid JSON is
// rejected before any write so the original file remains untouched.
func MergePluginConfig(configDir, bundle string, mode os.FileMode) error {
	cfgPath := filepath.Join(configDir, "opencode.json")
	cfg := map[string]any{}

	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("decode opencode.json: %w", err)
		}
		if cfg == nil {
			return fmt.Errorf("decode opencode.json: root must be an object")
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	abs, err := filepath.Abs(bundle)
	if err != nil {
		return err
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	cfg["plugin"] = []string{u.String()}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, mode)
}
