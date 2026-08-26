package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PrepareOfflinePluginDependency prevents OpenCode v1.18.11 from attempting a
// network dependency install before loading Codea's self-contained file plugin.
//
// OpenCode's config service always schedules @opencode-ai/plugin installation
// for writable config directories. When any plugin is configured, /agent waits
// for that background install to finish. In an offline environment this can
// block until the HTTP client times out even though Codea's bundle has no
// runtime npm dependency.
//
// The pinned v1.18.11 installer considers the dependency satisfied when
// node_modules exists and the root package-lock declares @opencode-ai/plugin.
// Codea owns this config directory, so seed exactly that local bookkeeping state
// before the Runtime starts. No package contents are needed because the Codea
// plugin bundle is fully self-contained.
func PrepareOfflinePluginDependency(configDir, version string) error {
	if strings.TrimSpace(configDir) == "" {
		return fmt.Errorf("config dir is required")
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("OpenCode plugin version is required")
	}

	if err := os.MkdirAll(filepath.Join(configDir, "node_modules"), 0o755); err != nil {
		return fmt.Errorf("create offline node_modules sentinel: %w", err)
	}

	lockPath := filepath.Join(configDir, "package-lock.json")
	lock := map[string]any{}
	if data, err := os.ReadFile(lockPath); err == nil {
		if err := json.Unmarshal(data, &lock); err != nil {
			return fmt.Errorf("decode package-lock.json: %w", err)
		}
		if lock == nil {
			return fmt.Errorf("decode package-lock.json: root must be an object")
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	lock["name"] = "codea-opencode-offline-runtime"
	lock["version"] = "0.0.0"
	lock["lockfileVersion"] = 3
	lock["requires"] = true

	packages, ok := lock["packages"].(map[string]any)
	if !ok {
		if lock["packages"] != nil {
			return fmt.Errorf("package-lock.json packages must be an object")
		}
		packages = map[string]any{}
		lock["packages"] = packages
	}
	root, ok := packages[""].(map[string]any)
	if !ok {
		if packages[""] != nil {
			return fmt.Errorf("package-lock.json root package must be an object")
		}
		root = map[string]any{}
		packages[""] = root
	}
	deps, ok := root["dependencies"].(map[string]any)
	if !ok {
		if root["dependencies"] != nil {
			return fmt.Errorf("package-lock.json root dependencies must be an object")
		}
		deps = map[string]any{}
		root["dependencies"] = deps
	}
	deps["@opencode-ai/plugin"] = version

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := lockPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, lockPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
