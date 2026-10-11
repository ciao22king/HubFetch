package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		provider string
		repoName string
	}{
		{"github https", "https://github.com/charmbracelet/bubbletea", "GitHub", "bubbletea"},
		{"github https with .git", "https://github.com/owner/project.git", "GitHub", "project"},
		{"github ssh", "git@github.com:ciao22king/GoFetch.git", "GitHub", "GoFetch"},
		{"github ssh scheme", "ssh://git@github.com/owner/project.git", "GitHub", "project"},
		{"gitlab nested group", "gitlab.com/team/tools/clone-kit.git", "GitLab", "clone-kit"},
		{"bare host prefix", "github.com/owner/project", "GitHub", "project"},
		{"self hosted generic", "https://git.example.dev/owner/project", "git.example.dev", "project"},
		{"www prefix stripped", "https://www.github.com/owner/project", "GitHub", "project"},
		{"trailing slash", "https://github.com/owner/project/", "GitHub", "project"},
		{"local file url", "file:///tmp/mirrors/project", "Local", "project"},
		{"local file url with .git", "file:///home/dev/project.git", "Local", "project"},
		{"windows file url", `file://C:\Users\runner\AppData\Local\Temp\repo\001`, "Local", "001"},
		{"windows file url forward slashes", "file://C:/Users/runner/repo", "Local", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRepository(tt.input)
			if err != nil {
				t.Fatalf("ParseRepository() error = %v", err)
			}
			if got.Provider != tt.provider || got.Name != tt.repoName {
				t.Fatalf("ParseRepository() = %#v, want provider %q name %q", got, tt.provider, tt.repoName)
			}
		})
	}
}

func TestParseRepositoryNormalizesWindowsFileURL(t *testing.T) {
	got, err := ParseRepository(`file://C:\Users\runner\AppData\Local\Temp\repo\001`)
	if err != nil {
		t.Fatalf("ParseRepository() error = %v", err)
	}
	want := "file:///C:/Users/runner/AppData/Local/Temp/repo/001"
	if got.URL != want {
		t.Fatalf("ParseRepository().URL = %q, want %q", got.URL, want)
	}
}

func TestParseRepositoryRejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", "   "},
		{"spaces", "https://github.com/owner/my project"},
		{"unsupported non domain", "https://localhost/owner/project"},
		{"missing repository", "https://github.com/owner"},
		{"ssh without path", "git@github.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseRepository(tt.input); err == nil {
				t.Fatalf("ParseRepository(%q) accepted invalid input", tt.input)
			}
		})
	}
}

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"project", "project"},
		{"my:project", "my-project"},
		{"a\\b/c", "a-b-c"},
		{"trailing.", "trailing"},
		{"CON", "CON-repo"},
		{"com1", "com1-repo"},
		{"...", "repository"},
	}
	for _, tt := range tests {
		if got := SanitizeRepoName(tt.input); got != tt.want {
			t.Fatalf("SanitizeRepoName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDestinationPathIsAbsoluteAndExpandsHome(t *testing.T) {
	got, err := DestinationPath("~", Repository{Name: "project"})
	if err != nil {
		t.Fatalf("DestinationPath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("DestinationPath() = %q, want an absolute path", got)
	}
	if filepath.Base(got) != "project" {
		t.Fatalf("DestinationPath() = %q, want it to end in project", got)
	}
	if runtime.GOOS != "windows" && filepath.Base(filepath.Dir(got)) == "~" {
		t.Fatalf("DestinationPath() did not expand the home directory: %q", got)
	}
}

func TestDestinationPathDefaultsToWorkingDirectory(t *testing.T) {
	got, err := DestinationPath("", Repository{Name: "project"})
	if err != nil {
		t.Fatalf("DestinationPath() error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("DestinationPath() = %q, want an absolute path", got)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	if got := ExpandHome("~/code"); got != filepath.Join(home, "code") {
		t.Fatalf("ExpandHome(~/code) = %q, want %q", got, filepath.Join(home, "code"))
	}
	if got := ExpandHome("/absolute/path"); got != "/absolute/path" {
		t.Fatalf("ExpandHome should leave absolute paths alone, got %q", got)
	}
}
