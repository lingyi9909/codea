package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const lockedOpenCodeVersion = "1.18.11"

// prepareOfflinePluginDependencies prevents OpenCode v1.18.11 from attempting
// npm installs before loading Codea's self-contained file:// plugin.
//
// OpenCode has two writable config dependency roots in Codea's supervised
// process: OPENCODE_CONFIG_DIR itself and Global.Path.config, which resolves to
// $XDG_CONFIG_HOME/opencode. Codea points XDG_CONFIG_HOME at
// <runtime-config>/xdg/config, so both roots must be primed. When an external
// plugin is configured, /agent waits for all of these dependency installers;
// leaving either root unprimed makes air-gapped Windows startup wait on npm.
func prepareOfflinePluginDependencies(cfgDir, openCodeVersion string) error {
	if openCodeVersion != lockedOpenCodeVersion {
		return fmt.Errorf("unsupported OpenCode dependency bootstrap version %q; locked=%s", openCodeVersion, lockedOpenCodeVersion)
	}
	roots := []string{
		cfgDir,
		filepath.Join(cfgDir, "xdg", "config", "opencode"),
	}
	for _, root := range roots {
		if err := prepareOfflinePluginDependencyDir(root, openCodeVersion); err != nil {
			return err
		}
	}
	return nil
}

func prepareOfflinePluginDependencyDir(dir, openCodeVersion string) error {
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		return fmt.Errorf("create runtime node_modules sentinel in %s: %w", dir, err)
	}

	pkg := struct {
		Private      bool              `json:"private"`
		Dependencies map[string]string `json:"dependencies"`
	}{
		Private: true,
		Dependencies: map[string]string{
			"@opencode-ai/plugin": openCodeVersion,
		},
	}
	if err := writeJSONFile(filepath.Join(dir, "package.json"), pkg); err != nil {
		return fmt.Errorf("write offline package.json in %s: %w", dir, err)
	}

	lock := struct {
		Name            string `json:"name"`
		LockfileVersion int    `json:"lockfileVersion"`
		Requires        bool   `json:"requires"`
		Packages        map[string]struct {
			Dependencies map[string]string `json:"dependencies,omitempty"`
		} `json:"packages"`
	}{
		Name:            "codea-runtime-config",
		LockfileVersion: 3,
		Requires:        true,
		Packages: map[string]struct {
			Dependencies map[string]string `json:"dependencies,omitempty"`
		}{
			"": {
				Dependencies: map[string]string{
					"@opencode-ai/plugin": openCodeVersion,
				},
			},
		},
	}
	if err := writeJSONFile(filepath.Join(dir, "package-lock.json"), lock); err != nil {
		return fmt.Errorf("write offline package-lock.json in %s: %w", dir, err)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
