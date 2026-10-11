package core

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Repository is a parsed clone target.
type Repository struct {
	URL      string // normalized URL git will actually fetch
	Provider string // GitHub / GitLab / Local / generic host
	Name     string // sanitized folder name
}

// unsafeNameChars matches every character Windows forbids in file names plus
// control characters. Compiling it once at package level keeps sanitization
// allocation-light on every keystroke of the destination preview.
var unsafeNameChars = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

// reservedWindowsNames are the device names Windows refuses as file names.
var reservedWindowsNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// invalidURLErr is shared so parse failures never allocate a new error per call.
var invalidURLErr = errors.New("URL non valido")

// SanitizeRepoName turns the repository segment of a URL into a folder name
// that is safe on Windows, macOS and Linux. Windows rules are the strictest,
// so applying them everywhere keeps clones portable across platforms.
func SanitizeRepoName(name string) string {
	name = strings.TrimSpace(name)
	name = unsafeNameChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, ". ")
	if name == "" {
		return "repository"
	}
	if base, _, _ := strings.Cut(strings.ToUpper(name), "."); reservedWindowsNames[base] {
		return name + "-repo"
	}
	return name
}

// ParseRepository understands the URL shapes people actually paste:
//
//	https://github.com/owner/repo(.git)
//	git@github.com:owner/repo(.git)
//	ssh://git@github.com/owner/repo(.git)
//	github.com/owner/repo
//	gitlab.com/group/subgroup/repo
//	file:///path/to/local/repo
//
// GitHub and GitLab are recognized explicitly; any other host that looks like
// a real domain is accepted as a generic Git remote so self-hosted instances
// and other forges keep working. file:// URLs are accepted as local remotes.
func ParseRepository(raw string) (Repository, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Repository{}, errors.New("incolla un URL GitHub o GitLab")
	}
	if strings.ContainsAny(raw, " \t\n") {
		return Repository{}, errors.New("l'URL non può contenere spazi")
	}

	normalized := raw
	if !strings.Contains(normalized, "://") && !strings.HasPrefix(normalized, "git@") {
		normalized = "https://" + normalized
	}

	var host, repoPath string
	switch {
	case strings.HasPrefix(normalized, "git@"):
		// SCP-like SSH URLs: git@github.com:owner/repository.git
		userHost, sshPath, found := strings.Cut(normalized, ":")
		if !found {
			return Repository{}, fmt.Errorf("%w: URL SSH incompleto", invalidURLErr)
		}
		host = strings.TrimPrefix(userHost, "git@")
		repoPath = sshPath
	default:
		parsed, err := url.Parse(normalized)
		if err != nil {
			return Repository{}, fmt.Errorf("%w: %v", invalidURLErr, err)
		}
		if parsed.Scheme == "file" {
			// Local repositories need no host validation: file:///path/to/repo is
			// always a valid generic remote, which also makes GoFetch handy for
			// local mirrors and tests.
			if parsed.Path == "" || parsed.Path == "/" {
				return Repository{}, errors.New("URL file:// non valido: percorso mancante")
			}
			name := SanitizeRepoName(strings.TrimSuffix(path.Base(parsed.Path), ".git"))
			return Repository{URL: normalized, Provider: "Local", Name: name}, nil
		}
		host = parsed.Hostname()
		repoPath = parsed.Path
	}

	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	if host == "" {
		return Repository{}, fmt.Errorf("%w: host mancante", invalidURLErr)
	}
	// Obvious non-hosts (localhost, example.*) are rejected so typos fail fast
	// instead of turning into a confusing clone error.
	if !looksLikeHost(host) {
		return Repository{}, fmt.Errorf("provider non supportato: %s (usa github.com o gitlab.com)", host)
	}

	repoPath = strings.Trim(repoPath, "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	segments := strings.Split(repoPath, "/")
	if len(segments) < 2 || segments[0] == "" || segments[len(segments)-1] == "" {
		return Repository{}, fmt.Errorf("serve un URL nel formato %s/owner/repository", host)
	}

	return Repository{
		URL:      normalized,
		Provider: Provider(host),
		Name:     SanitizeRepoName(segments[len(segments)-1]),
	}, nil
}

func looksLikeHost(host string) bool {
	switch host {
	case "localhost", "example.com", "example.org", "example.net":
		return false
	default:
		return strings.Contains(host, ".")
	}
}

// DestinationPath returns an absolute clone target so a later change of the
// process working directory can never move where git actually writes.
func DestinationPath(parent string, repo Repository) (string, error) {
	parent = ExpandHome(strings.TrimSpace(parent))
	if parent == "" {
		parent = "."
	}
	if strings.HasPrefix(parent, "~/") {
		return "", fmt.Errorf("percorso home non espanso: %s", parent)
	}
	absolute, err := filepath.Abs(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(absolute, repo.Name), nil
}

// NonEmptyDir reports whether path exists and already contains entries, which
// would make git refuse to clone into it. A single ReadDir call answers both
// questions, which keeps this cheap enough to run on every Enter press.
func NonEmptyDir(p string) bool {
	entries, err := os.ReadDir(p)
	return err == nil && len(entries) > 0
}

// ExpandHome resolves a leading ~ against the user's home directory.
func ExpandHome(value string) string {
	value = strings.TrimSpace(value)
	if value == "~" || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return value
		}
		if value == "~" {
			return home
		}
		return filepath.Join(home, value[2:])
	}
	return value
}
