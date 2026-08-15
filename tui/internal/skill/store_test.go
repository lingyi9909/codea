package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "skills.json")
	s := NewFileStore(path)
	overrides := map[string]bool{"git": true, "unit-test": false}

	if err := s.Save(overrides); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["git"] != true || got["unit-test"] != false {
		t.Fatalf("round-trip mismatch: %v", got)
	}
}

func TestFileStoreLoadMissing(t *testing.T) {
	s := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty overrides, got %v", got)
	}
}

func TestFileStoreLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewFileStore(path)
	if _, err := s.Load(); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
