//go:build windows

package checkpoint

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createJunction(t *testing.T, link, target string) {
	t.Helper()
	cmd := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create junction %q -> %q: %v\n%s", link, target, err, output)
	}
}

func TestRestoreBlocksJunctionAncestorEscape(t *testing.T) {
	t.Run("write", func(t *testing.T) {
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
		createJunction(t, filepath.Join(project, "src"), outside)

		result, err := s.Restore(context.Background(), baseline.ID)
		if err == nil || !IsCode(err, CodeRestoreInterrupted) {
			t.Fatalf("expected restore to fail closed on junction ancestor, result=%+v err=%v", result, err)
		}
		if got := readText(t, outside, "sentinel.txt"); got != "outside-safe" {
			t.Fatalf("outside sentinel changed: %q", got)
		}
		if _, statErr := os.Stat(filepath.Join(outside, "A.java")); !os.IsNotExist(statErr) {
			t.Fatalf("outside file must not be created, err=%v", statErr)
		}
	})

	t.Run("delete", func(t *testing.T) {
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
		createJunction(t, filepath.Join(project, "src"), outside)

		changed, err := s.applyRestore(context.Background(), empty, withFile)
		if err == nil {
			t.Fatalf("expected delete restore to fail closed on junction ancestor, changed=%d", changed)
		}
		if changed != 0 {
			t.Fatalf("files changed=%d, want 0 before unsafe deletion", changed)
		}
		if got := readText(t, outside, "delete.txt"); got != "outside-safe" {
			t.Fatalf("outside file was deleted or changed: %q", got)
		}
	})
}
