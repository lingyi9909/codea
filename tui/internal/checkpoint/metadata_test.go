package checkpoint

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetadataCorruptionFailsClosed(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	s, err := NewService(context.Background(), home, project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(s.paths.Metadata), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("{ definitely not json")
	if err := os.WriteFile(s.paths.Metadata, original, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err == nil || !IsCode(err, CodeStateCorrupt) {
		t.Fatalf("expected CHECKPOINT_STATE_CORRUPT, got %v", err)
	}
	after, readErr := os.ReadFile(s.paths.Metadata)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(original) {
		t.Fatalf("corrupt history was silently replaced: %q", after)
	}
}

func TestMetadataWriteLeavesNoTemporaryFile(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Dir(s.paths.Metadata))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Fatalf("atomic metadata temp leaked: %s", entry.Name())
		}
	}
}
