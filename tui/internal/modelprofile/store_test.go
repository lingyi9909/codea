package modelprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func TestStoreKeysProviderAndModel(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	first := completeProfile(runtime.ModelRef{ProviderID: "provider-a", ModelID: "same-model"}, CapabilityStrong)
	second := completeProfile(runtime.ModelRef{ProviderID: "provider-b", ModelID: "same-model"}, CapabilityMedium)
	if err := store.Save(first); err != nil { t.Fatal(err) }
	if err := store.Save(second); err != nil { t.Fatal(err) }
	gotFirst, ok, err := store.Load(first.Model)
	if err != nil || !ok { t.Fatalf("load first ok=%v err=%v", ok, err) }
	gotSecond, ok, err := store.Load(second.Model)
	if err != nil || !ok { t.Fatalf("load second ok=%v err=%v", ok, err) }
	if gotFirst.Overall != CapabilityStrong || gotSecond.Overall != CapabilityMedium {
		t.Fatalf("profiles collided: first=%s second=%s", gotFirst.Overall, gotSecond.Overall)
	}
	if ProfilePath(home, first.Model) == ProfilePath(home, second.Model) {
		t.Fatal("provider must participate in profile key")
	}
}

func TestStoreAtomicReloadAndUnicodeHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "Codea Home 中文")
	store := NewStore(home)
	want := completeProfile(runtime.ModelRef{ProviderID: "private", ModelID: "model 7"}, CapabilityStrong)
	if err := store.Save(want); err != nil { t.Fatal(err) }
	got, ok, err := store.Load(want.Model)
	if err != nil || !ok { t.Fatalf("load ok=%v err=%v", ok, err) }
	if got.Model != want.Model || got.Overall != want.Overall || got.Version != ProfileVersion {
		t.Fatalf("roundtrip mismatch: %#v", got)
	}
	entries, err := os.ReadDir(filepath.Dir(ProfilePath(home, want.Model)))
	if err != nil { t.Fatal(err) }
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") { t.Fatalf("temporary file leaked: %s", entry.Name()) }
	}
}

func TestStoreCorruptAndStaleProfiles(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home)
	ref := runtime.ModelRef{ProviderID: "private", ModelID: "model"}
	path := ProfilePath(home, ref)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil { t.Fatal(err) }
	if _, _, err := store.Load(ref); err == nil || !strings.Contains(err.Error(), CodeModelProfileCorrupt) {
		t.Fatalf("corrupt profile err=%v", err)
	}
	stale := `{"model":{"ProviderID":"private","ModelID":"model"},"toolCalling":"strong","structuredOutput":"strong","patchFollowing":"strong","planning":"strong","overall":"strong","qualifiedAt":"2026-09-06T00:00:00Z","version":999}`
	if err := os.WriteFile(path, []byte(stale), 0o600); err != nil { t.Fatal(err) }
	_, ok, err := store.Load(ref)
	if err != nil || ok { t.Fatalf("stale profile should be unqualified: ok=%v err=%v", ok, err) }
}

func TestAggregateProfileConservatively(t *testing.T) {
	cases := []struct {
		name string
		tool, structured, patch, planning CapabilityLevel
		want CapabilityLevel
	}{
		{"all strong", CapabilityStrong, CapabilityStrong, CapabilityStrong, CapabilityStrong, CapabilityStrong},
		{"medium dimension", CapabilityStrong, CapabilityMedium, CapabilityStrong, CapabilityStrong, CapabilityMedium},
		{"weak critical tool", CapabilityWeak, CapabilityStrong, CapabilityStrong, CapabilityStrong, CapabilityWeak},
		{"weak other dimension", CapabilityStrong, CapabilityStrong, CapabilityWeak, CapabilityStrong, CapabilityWeak},
		{"unknown incomplete", CapabilityStrong, CapabilityUnknown, CapabilityStrong, CapabilityStrong, CapabilityUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Aggregate(tc.tool, tc.structured, tc.patch, tc.planning); got != tc.want {
				t.Fatalf("Aggregate()=%s want=%s", got, tc.want)
			}
		})
	}
}

func TestStoreRejectsIncompleteProfile(t *testing.T) {
	store := NewStore(t.TempDir())
	profile := completeProfile(runtime.ModelRef{ProviderID: "p", ModelID: "m"}, CapabilityStrong)
	profile.Planning = CapabilityUnknown
	profile.Overall = Aggregate(profile.ToolCalling, profile.StructuredOutput, profile.PatchFollowing, profile.Planning)
	if err := store.Save(profile); err == nil {
		t.Fatal("incomplete qualification must not persist")
	}
}

func completeProfile(ref runtime.ModelRef, level CapabilityLevel) ModelCapabilityProfile {
	return ModelCapabilityProfile{
		Model: ref,
		ToolCalling: level,
		StructuredOutput: level,
		PatchFollowing: level,
		Planning: level,
		Overall: level,
		QualifiedAt: time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC),
		Version: ProfileVersion,
	}
}
