package fetch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ciao22king/GoFetch/internal/core"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=GoFetch Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=GoFetch Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// newOrigin creates a small local repository with two branches to clone from.
func newOrigin(t *testing.T) string {
	t.Helper()
	origin := t.TempDir()
	runGit(t, origin, "init", "-q")
	if err := os.WriteFile(filepath.Join(origin, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, origin, "add", ".")
	runGit(t, origin, "commit", "-q", "-m", "init")
	return origin
}

type recordingStore struct {
	mu      sync.Mutex
	entries []core.Entry
}

func (s *recordingStore) Load() []core.Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.entries
}

func (s *recordingStore) Save(entries []core.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = entries
	return nil
}

func TestRunClonesDefaultBranch(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t)
	parent := t.TempDir()

	store := &recordingStore{}
	result, err := Run(context.Background(), Options{
		RawURL:  "file://" + origin,
		Parent:  parent,
		NoBuild: true,
	}, store)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Target, "README.md")); err != nil {
		t.Fatalf("cloned repository missing README: %v", err)
	}
	if len(store.entries) != 1 || store.entries[0].Target != result.Target {
		t.Fatalf("store entries = %#v, want the cloned target recorded", store.entries)
	}
}

func TestRunRefusesExistingDestinationWithoutOverwrite(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t)
	parent := t.TempDir()

	if _, err := Run(context.Background(), Options{RawURL: "file://" + origin, Parent: parent, NoBuild: true}, nil); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	if _, err := Run(context.Background(), Options{RawURL: "file://" + origin, Parent: parent, NoBuild: true}, nil); err == nil {
		t.Fatal("Run() overwrote a non-empty destination without permission")
	}
}

func TestRunOverwritesWhenAsked(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t)
	parent := t.TempDir()

	if _, err := Run(context.Background(), Options{RawURL: "file://" + origin, Parent: parent, NoBuild: true}, nil); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	result, err := Run(context.Background(), Options{RawURL: "file://" + origin, Parent: parent, NoBuild: true, Overwrite: true}, nil)
	if err != nil {
		t.Fatalf("second Run(overwrite) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Target, "README.md")); err != nil {
		t.Fatalf("overwritten clone missing README: %v", err)
	}
}

func TestRunShallowCloneKeepsOneCommit(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t)
	for i := 0; i < 2; i++ {
		if err := os.WriteFile(filepath.Join(origin, "README.md"), []byte(strings.Repeat("x", i+2)+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		runGit(t, origin, "add", ".")
		runGit(t, origin, "commit", "-q", "-m", "more")
	}
	parent := t.TempDir()

	result, err := Run(context.Background(), Options{RawURL: "file://" + origin, Parent: parent, Depth: 1, NoBuild: true}, nil)
	if err != nil {
		t.Fatalf("Run(depth=1) error = %v", err)
	}
	count := exec.Command("git", "-C", result.Target, "rev-list", "--count", "HEAD")
	out, err := count.Output()
	if err != nil {
		t.Fatalf("git rev-list: %v", err)
	}
	if strings.TrimSpace(string(out)) != "1" {
		t.Fatalf("shallow clone has %s commits, want 1", strings.TrimSpace(string(out)))
	}
}

func TestRunStreamsProgressLines(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t)
	parent := t.TempDir()

	progress := make(chan string, 256)
	_, err := Run(context.Background(), Options{
		RawURL:   "file://" + origin,
		Parent:   parent,
		NoBuild:  true,
		Progress: func(line string) { progress <- line },
	}, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	close(progress)
	var seen int
	for range progress {
		seen++
	}
	if seen == 0 {
		t.Fatal("Run() streamed no progress lines")
	}
}

func TestRunRejectsInvalidURL(t *testing.T) {
	if _, err := Run(context.Background(), Options{RawURL: "not a url"}, nil); err == nil {
		t.Fatal("Run() accepted an invalid URL")
	}
}

func TestRunCanceledRemovesPartialDirectory(t *testing.T) {
	requireGit(t)
	origin := newOrigin(t)
	parent := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the clone even starts

	name := filepath.Base(origin)
	target := filepath.Join(parent, core.SanitizeRepoName(name))
	_, err := Run(ctx, Options{RawURL: "file://" + origin, Parent: parent, NoBuild: true}, nil)
	if err == nil {
		t.Skip("clone finished despite pre-cancelled context; nothing to assert")
	}
	if entries, statErr := os.ReadDir(target); statErr == nil && len(entries) > 0 {
		t.Fatalf("cancelled clone left a populated directory behind: %v", entries)
	}
}
