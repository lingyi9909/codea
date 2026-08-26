package opencode

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	cfg["plugin"] = []string{pluginFileURL(abs)}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, data, mode); err != nil {
		return err
	}

	// OpenCode v1.18.11 schedules @opencode-ai/plugin installation for every
	// config directory and /agent waits for those background installs whenever a
	// plugin is configured. Codea runs with XDG_CONFIG_HOME=<configDir>/xdg/config,
	// so both Global.Path.config and OPENCODE_CONFIG_DIR are distinct writable
	// directories. Seed both Codea-owned locations before Runtime startup so a
	// machine without npm/internet never blocks here.
	for _, dir := range []string{
		filepath.Join(configDir, "xdg", "config", "opencode"),
		configDir,
	} {
		if err := PrepareOfflinePluginDependency(dir, pinnedOpenCodePluginVersion); err != nil {
			return fmt.Errorf("prepare offline OpenCode plugin dependency in %s: %w", dir, err)
		}
	}
	return nil
}

// pluginFileURL converts a local filesystem path into a standards-compliant
// file URL on every host. In particular, a Windows drive path must be encoded
// as file:///C:/... rather than file://C:/..., where C: would incorrectly be
// interpreted as the URL authority/host. The conversion is intentionally
// platform-independent so Windows semantics stay covered by CI on non-Windows
// runners too.
func pluginFileURL(localPath string) string {
	slash := strings.ReplaceAll(localPath, `\`, "/")

	// Windows local drive: C:/x -> /C:/x so net/url emits file:///C:/x.
	if len(slash) >= 3 && isASCIIAlpha(slash[0]) && slash[1] == ':' && slash[2] == '/' {
		slash = "/" + slash
	}

	// UNC path: //server/share/x -> file://server/share/x.
	if strings.HasPrefix(slash, "//") {
		rest := strings.TrimPrefix(slash, "//")
		if host, pathPart, ok := strings.Cut(rest, "/"); ok && host != "" {
			return (&url.URL{Scheme: "file", Host: host, Path: "/" + pathPart}).String()
		}
	}

	return (&url.URL{Scheme: "file", Path: slash}).String()
}

func isASCIIAlpha(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
