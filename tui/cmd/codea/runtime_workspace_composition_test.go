package main

import (
	"path/filepath"
	"testing"

	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask23SharedDoctorFactoryBuildsServiceForExistingRuntime(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEA_HOME", home)
	t.Setenv("CODEA_RUNTIME_CONFIG_DIR", filepath.Join(home, "runtime-config"))

	svc, err := newDoctorService(fakeruntime.New(), nil, "http://127.0.0.1:4141")
	if err != nil {
		t.Fatalf("newDoctorService: %v", err)
	}
	if svc == nil {
		t.Fatal("newDoctorService returned nil service")
	}
}
