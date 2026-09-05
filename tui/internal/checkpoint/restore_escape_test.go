//go:build !windows

package checkpoint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreBlocksSymlinkAncestorEscapeWrite(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()
	writeText(t, project, "src/A.java", "baseline")
	writeText(t, outside, "sentinel.txt", "outside-safe")

	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(project, "src")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, "src")); err != nil {
		t.Fatal(err)
	}

	result, err := s.Restore(context.Background(), baseline.ID)
	if err == nil || !IsCode(err, CodeRestoreInterrupted) {
		t.Fatalf("expected restore to fail closed on symlink ancestor, result=%+v err=%v", result, err)
	}
	if got := readText(t, outside, "sentinel.txt"); got != "outside-safe" {
		t.Fatalf("outside sentinel changed: %q", got)
	}
	if _, statErr := os.Stat(filepath.Join(outside, "A.java")); !os.IsNotExist(statErr) {
		t.Fatalf("outside file must not be created, err=%v", statErr)
	}
}

func TestApplyRestoreBlocksSymlinkAncestorEscapeDelete(t *testing.T) {
	project := t.TempDir()
	outside := t.TempDir()
	writeText(t, project, "keep.txt", "keep")
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	empty, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, project, "src/delete.txt", "inside")
	withFile, err := s.Create(context.Background(), CreateRequest{Kind: KindManual})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(project, "src")); err != nil {
		t.Fatal(err)
	}
	writeText(t, outside, "delete.txt", "outside-safe")
	if err := os.Symlink(outside, filepath.Join(project, "src")); err != nil {
		t.Fatal(err)
	}

	changed, err := s.applyRestore(context.Background(), empty, withFile)
	if err == nil {
		t.Fatalf("expected delete restore to fail closed on symlink ancestor, changed=%d", changed)
	}
	if changed != 0 {
		t.Fatalf("files changed=%d, want 0 before unsafe deletion", changed)
	}
	if got := readText(t, outside, "delete.txt"); got != "outside-safe" {
		t.Fatalf("outside file was deleted or changed: %q", got)
	}
}

func TestSafetySkippedDirectoryProtectsDescendants(t *testing.T) {
	project := t.TempDir()
	writeText(t, project, "keep.txt", "keep")
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	empty, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, project, "src/A.java", "target")
	target, err := s.Create(context.Background(), CreateRequest{Kind: KindManual})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(project, "src")); err != nil {
		t.Fatal(err)
	}

	safety := empty
	safety.Skipped = []SkippedPath{{Path: "src", Reason: "symlink"}}
	changed, err := s.applyRestore(context.Background(), target, safety)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 0 {
		t.Fatalf("files changed=%d, want 0 for descendant of skipped directory", changed)
	}
	if _, statErr := os.Stat(filepath.Join(project, "src", "A.java")); !os.IsNotExist(statErr) {
		t.Fatalf("skipped directory descendant must remain untouched, err=%v", statErr)
	}
}
