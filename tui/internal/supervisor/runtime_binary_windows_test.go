//go:build windows

package supervisor

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestConfigureProcessRemovesRuntimeZoneIdentifier(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "opencode.exe")
	if err := os.WriteFile(binary, []byte("test-binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	zone := binary + ":Zone.Identifier"
	if err := os.WriteFile(zone, []byte("[ZoneTransfer]\r\nZoneId=3\r\n"), 0o600); err != nil {
		t.Fatalf("create Zone.Identifier ADS: %v", err)
	}
	if _, err := os.Stat(zone); err != nil {
		t.Fatalf("Zone.Identifier should exist before process configuration: %v", err)
	}

	cmd := exec.Command(binary)
	configureProcess(cmd)

	if _, err := os.Stat(zone); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Zone.Identifier still exists after process configuration: %v", err)
	}
}
