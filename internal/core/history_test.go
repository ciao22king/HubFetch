package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpsertMovesExistingTargetToFront(t *testing.T) {
	older := Entry{
		Repository: Repository{Name: "older"},
		Target:     filepath.Join("/tmp", "older"),
		FetchedAt:  time.Now().Add(-time.Hour),
	}
	current := Entry{
		Repository: Repository{Name: "current"},
		Target:     filepath.Join("/tmp", "current"),
	}

	entries := Upsert([]Entry{older}, current)
	entries = Upsert(entries, older)

	if len(entries) != 2 {
		t.Fatalf("Upsert() returned %d entries, want 2", len(entries))
	}
	if entries[0].Name != "older" {
		t.Fatalf("Upsert() first entry = %q, want older", entries[0].Name)
	}
	if !entries[0].FetchedAt.After(older.FetchedAt) {
		t.Fatal("Upsert() did not refresh fetched_at")
	}
}

func TestUpsertCapsEntries(t *testing.T) {
	entries := make([]Entry, 0, MaxHistoryEntries)
	for i := 0; i < MaxHistoryEntries+20; i++ {
		entries = Upsert(entries, Entry{
			Repository: Repository{Name: "repo"},
			Target:     filepath.Join("/tmp", "repo", string(rune('a'+i%26))+string(rune('0'+i%10))),
		})
	}
	if len(entries) > MaxHistoryEntries {
		t.Fatalf("Upsert() kept %d entries, want at most %d", len(entries), MaxHistoryEntries)
	}
}

func TestRemoveEntry(t *testing.T) {
	entries := []Entry{
		{Target: filepath.Join("/tmp", "a")},
		{Target: filepath.Join("/tmp", "b")},
	}
	got := Remove(entries, filepath.Join("/tmp", "a"))
	if len(got) != 1 || got[0].Target != filepath.Join("/tmp", "b") {
		t.Fatalf("Remove() = %#v, want only b", got)
	}
}

func TestSearch(t *testing.T) {
	entries := []Entry{
		{Repository: Repository{Name: "bubbletea", Provider: "GitHub", URL: "https://github.com/charmbracelet/bubbletea"}, Target: "/code/bubbletea"},
		{Repository: Repository{Name: "clone-kit", Provider: "GitLab", URL: "https://gitlab.com/team/clone-kit"}, Target: "/work/clone-kit"},
	}

	if got := Search(entries, ""); len(got) != 2 {
		t.Fatalf("Search(\"\") = %v, want both entries", got)
	}
	if got := Search(entries, "gitlab"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("Search(gitlab) = %v, want [1]", got)
	}
	if got := Search(entries, "/work"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("Search(/work) = %v, want [1]", got)
	}
	if got := Search(entries, "missing"); len(got) != 0 {
		t.Fatalf("Search(missing) = %v, want none", got)
	}
}

func TestRecentDirectories(t *testing.T) {
	entries := []Entry{
		{Target: filepath.Join("/code", "a")},
		{Target: filepath.Join("/code", "b")},
		{Target: filepath.Join("/work", "c")},
	}
	got := RecentDirectories(entries, 8)
	if len(got) != 2 {
		t.Fatalf("RecentDirectories() = %v, want 2 unique directories", got)
	}
}

func TestRelativeTime(t *testing.T) {
	if got := RelativeTime(time.Time{}); got != "" {
		t.Fatalf("RelativeTime(zero) = %q, want empty", got)
	}
	if got := RelativeTime(time.Now()); got != "adesso" {
		t.Fatalf("RelativeTime(now) = %q, want adesso", got)
	}
	if got := RelativeTime(time.Now().Add(-2 * time.Hour)); got != "2h fa" {
		t.Fatalf("RelativeTime(-2h) = %q, want 2h fa", got)
	}
}

func TestStoreSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	entries := []Entry{{
		Repository: Repository{URL: "https://github.com/owner/project", Provider: "GitHub", Name: "project"},
		Target:     filepath.Join(dir, "project"),
		FetchedAt:  time.Now(),
	}}
	if err := store.Save(entries); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}

	loaded := store.Load()
	if len(loaded) != 1 || loaded[0].Name != "project" {
		t.Fatalf("store.Load() = %#v, want the saved entry", loaded)
	}

	path := filepath.Join(dir, "history.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("history file not written: %v", err)
	}
	// The temp file used for the atomic write must not be left behind.
	matches, _ := filepath.Glob(filepath.Join(dir, "history-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary history files left behind: %v", matches)
	}
}

func TestStoreLoadMissingFileDegradesToEmpty(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "nope"))
	if got := store.Load(); len(got) != 0 {
		t.Fatalf("Load() on a missing file = %v, want empty", got)
	}
}

func TestStoreLoadCorruptFileDegradesToEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "history.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	// A fresh cache miss must read the corrupt file and degrade gracefully.
	store.cache = nil
	if got := store.Load(); len(got) != 0 {
		t.Fatalf("Load() on a corrupt file = %v, want empty", got)
	}
}

func TestEntryLabelFallsBackToFolderName(t *testing.T) {
	entry := Entry{Target: filepath.Join("/code", "alpha")}
	if entry.Label() != "alpha" {
		t.Fatalf("Label() = %q, want alpha", entry.Label())
	}
	entry.Repository.Name = "renamed"
	if entry.Label() != "renamed" {
		t.Fatalf("Label() = %q, want renamed", entry.Label())
	}
}

func TestSameTargetCleansPaths(t *testing.T) {
	if !SameTarget("/code/repo/", "/code/repo") {
		t.Fatal("trailing separator should not change identity")
	}
	if SameTarget("/code/repo", "/code/other") {
		t.Fatal("different folders are not the same target")
	}
}

func TestMatchesIsCaseInsensitiveOnFields(t *testing.T) {
	entry := Entry{Repository: Repository{Name: "BubbleTea", Provider: "GitHub"}, Target: "/Code/BubbleTea"}
	if !strings.Contains(strings.ToLower(entry.Name), "bubbletea") {
		t.Fatal("sanity check failed")
	}
	indexes := Search([]Entry{entry}, "BUBBLE")
	if len(indexes) != 1 {
		t.Fatalf("Search(BUBBLE) = %v, want one match", indexes)
	}
}
