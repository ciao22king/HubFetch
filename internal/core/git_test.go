package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func TestParseRemoteBranches(t *testing.T) {
	output := "65049fb8fccea09cc947147362a92031dffd9bf8\trefs/heads/main\n" +
		"f59319031bf01430d0581d0ad931a9866044f22c\trefs/heads/feat/login\n" +
		"fd51812edfc12a889ebc8023c727bdb1445f8bae\trefs/tags/v1.0.0\n" +
		"\n"
	got := ParseRemoteBranches(output)
	want := []string{"main", "feat/login"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseRemoteBranches() = %v, want %v", got, want)
	}
}

func TestParseRemoteBranchesEmpty(t *testing.T) {
	if got := ParseRemoteBranches(""); got != nil {
		t.Fatalf("ParseRemoteBranches(\"\") = %v, want nil", got)
	}
}

func TestBranchCompletions(t *testing.T) {
	branches := []string{"main", "develop", "feat/login", "feat/logout", "fix/crash"}

	tests := []struct {
		prefix string
		want   []string
	}{
		{"", branches},
		{"feat/", []string{"feat/login", "feat/logout"}},
		{"main", []string{"main"}},
		{"  ", branches},
	}
	for _, tt := range tests {
		got := BranchCompletions(branches, tt.prefix)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("BranchCompletions(%q) = %v, want %v", tt.prefix, got, tt.want)
		}
	}
	if got := BranchCompletions(branches, "nomatch"); len(got) != 0 {
		t.Errorf("BranchCompletions(\"nomatch\") = %v, want empty", got)
	}
}

func TestFriendlyCloneError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"not found", "remote: Repository not found.", "non trovato"},
		{"auth", "fatal: Authentication failed for 'https://example.com'", "autenticazione"},
		{"dns", "fatal: unable to access: Could not resolve host: github.com", "raggiungere"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FriendlyCloneError(ErrCloneCanceled, tt.output, "/tmp/x")
			if err == nil || !containsFold(err.Error(), tt.want) {
				t.Fatalf("FriendlyCloneError() = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestFriendlyCloneErrorNil(t *testing.T) {
	if err := FriendlyCloneError(nil, "anything", "/tmp/x"); err != nil {
		t.Fatalf("FriendlyCloneError(nil) = %v, want nil", err)
	}
}

func TestNonEmptyDir(t *testing.T) {
	empty := t.TempDir()
	if NonEmptyDir(empty) {
		t.Fatalf("NonEmptyDir() = true for an empty directory")
	}

	populated := t.TempDir()
	if err := os.WriteFile(filepath.Join(populated, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !NonEmptyDir(populated) {
		t.Fatalf("NonEmptyDir() = false for a populated directory")
	}

	if NonEmptyDir(filepath.Join(empty, "missing")) {
		t.Fatalf("NonEmptyDir() = true for a missing directory")
	}
}

func TestSummarizePull(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"empty", "", "Repository aggiornato."},
		{"up to date", "Already up to date.", "Già aggiornato: nessuna novità dal remote."},
		{"up-to-date variant", "Already up-to-date.", "Già aggiornato: nessuna novità dal remote."},
		{"updated", "Updating cfcd826..50d2057\nFast-forward\n main.go | 10 +++++-----", "Aggiornato: Updating cfcd826..50d2057"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SummarizePull(tt.output); got != tt.want {
				t.Errorf("SummarizePull(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestFriendlyPullError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"not a repo", "fatal: not a git repository", "non è un repository Git"},
		{"diverged", "fatal: Not possible to fast-forward, aborting.", "modifiche divergenti"},
		{"auth", "fatal: Authentication failed", "autenticazione richiesta"},
		{"network", "fatal: unable to access 'https://x': Could not resolve host", "controlla la connessione"},
		{"local changes", "error: Your local changes to the following files would be overwritten by merge", "modifiche locali"},
		{"fallback", "something unexpected", "git pull non riuscito"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FriendlyPullError(os.ErrPermission, tt.output)
			if err == nil || !contains(err.Error(), tt.want) {
				t.Errorf("FriendlyPullError(%q) = %v, want substring %q", tt.output, err, tt.want)
			}
		})
	}
}
