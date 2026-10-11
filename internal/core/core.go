// Package core holds the domain logic of GoFetch: URL parsing, destination
// computation, history storage, build detection and Git command helpers. It
// has no UI dependencies so it can be reused by both the TUI and headless
// mode — and tested — without dragging a terminal along.
package core

import "strings"

// Provider maps well-known hosts to their display name. Self-hosted servers
// are identified by their own hostname (e.g. "git.example.dev") so the UI
// and the history can tell them apart from github.com/gitlab.com clones.
func Provider(host string) string {
	switch host {
	case "github.com":
		return "GitHub"
	case "gitlab.com":
		return "GitLab"
	default:
		return host
	}
}

// friendlyRule maps a group of git-output keywords (matched case-insensitively)
// to a short actionable Italian hint.
type friendlyRule struct {
	keywords []string
	build    func() error
}

// friendlyError applies rules to git's output and returns the first match, or
// nil when nothing matches. Shared by clone and pull so the same network and
// authentication hints stay consistent everywhere.
func friendlyError(err error, output string, rules []friendlyRule) error {
	lower := strings.ToLower(output)
	for _, rule := range rules {
		for _, keyword := range rule.keywords {
			if strings.Contains(lower, keyword) {
				return rule.build()
			}
		}
	}
	return nil
}

// authRule, networkRule and similar are reused by both FriendlyCloneError and
// FriendlyPullError.
var (
	authRule = friendlyRule{
		keywords: []string{"authentication failed", "permission denied", "terminal prompts disabled"},
		build: func() error {
			return errText("autenticazione richiesta: configura una chiave SSH o un token Git")
		},
	}
	networkRule = friendlyRule{
		keywords: []string{"could not resolve host", "unable to access", "network is unreachable"},
		build: func() error {
			return errText("impossibile raggiungere il server: controlla la connessione")
		},
	}
)

func errText(message string) error { return &simpleError{text: message} }

type simpleError struct{ text string }

func (e *simpleError) Error() string { return e.text }
