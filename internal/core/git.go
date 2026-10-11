package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// GitCommand builds a git invocation with the environment every GoFetch
// operation needs: GIT_TERMINAL_PROMPT=0 guarantees git fails fast instead of
// blocking forever on an interactive credential prompt.
func GitCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd
}

// CloneOptions describes one `git clone` run.
type CloneOptions struct {
	Repo     Repository
	Target   string
	Branch   string            // empty = remote default branch
	Depth    int               // > 0 = shallow clone limited to that many commits
	Progress func(line string) // optional live output callback
}

// Clone runs `git clone --progress`, streaming each output line to
// opts.Progress (when set) and capturing everything for error reporting. It
// returns git's combined output together with the mapped friendly error.
func Clone(ctx context.Context, opts CloneOptions) (output string, err error) {
	args := []string{"clone", "--progress"}
	if branch := strings.TrimSpace(opts.Branch); branch != "" {
		args = append(args, "--branch", branch)
	}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}
	args = append(args, opts.Repo.URL, opts.Target)

	cmd := GitCommand(ctx, args...)

	// git writes clone progress to stderr. Drain stdout+stderr through one
	// pipe scanned on a separate goroutine so a slow consumer can never block
	// git, and publish each line both live and to the capture buffer.
	var combined bytes.Buffer
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	lines := make(chan struct{})
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			combined.WriteString(line)
			combined.WriteByte('\n')
			if opts.Progress != nil {
				opts.Progress(line)
			}
		}
	}()

	runErr := cmd.Run()
	writer.Close()
	<-lines

	text := combined.String()
	if ctx.Err() == context.Canceled {
		return text, ErrCloneCanceled
	}
	return text, FriendlyCloneError(runErr, text, opts.Target)
}

// ErrCloneCanceled marks a clone aborted by the user (Esc in the TUI).
var ErrCloneCanceled = errText("clone annullato")

// FriendlyCloneError replaces git's wall of text with a short, actionable hint
// while preserving the raw output for anyone who needs to debug it.
func FriendlyCloneError(err error, output, target string) error {
	if err == nil {
		return nil
	}
	if matched := friendlyError(err, output, cloneRules(target)); matched != nil {
		return matched
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return errText("git clone non riuscito")
	}
	return fmt.Errorf("git clone non riuscito: %s", message)
}

func cloneRules(target string) []friendlyRule {
	return []friendlyRule{
		{keywords: []string{"repository not found"}, build: func() error {
			return errText("repository non trovato o privato: controlla l'URL e le tue credenziali")
		}},
		{keywords: []string{"remote branch", "not found"}, build: func() error {
			return errText("branch remoto inesistente: controlla il nome del branch")
		}},
		{keywords: []string{"could not read from remote repository"}, build: authRule.build},
		authRule,
		networkRule,
		{keywords: []string{"already exists and is not an empty directory"}, build: func() error {
			return fmt.Errorf("la cartella %s esiste già e non è vuota", target)
		}},
	}
}

// RemoteBranches lists the branches advertised by a remote without cloning
// anything: `git ls-remote --heads` only reads refs, so it is fast and needs
// no working tree.
func RemoteBranches(ctx context.Context, repoURL string) ([]string, error) {
	cmd := GitCommand(ctx, "ls-remote", "--heads", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return ParseRemoteBranches(string(output)), nil
}

// ParseRemoteBranches extracts branch names from `git ls-remote --heads`
// output, whose lines look like:
//
//	65049fb8fccea09cc947147362a92031dffd9bf8        refs/heads/main
func ParseRemoteBranches(output string) []string {
	var branches []string
	start := 0
	for start <= len(output) {
		var line string
		if end := strings.IndexByte(output[start:], '\n'); end >= 0 {
			line = output[start : start+end]
			start += end + 1
		} else {
			line = output[start:]
			start = len(output) + 1
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if name, ok := strings.CutPrefix(fields[1], "refs/heads/"); ok && name != "" {
			branches = append(branches, name)
		}
	}
	return branches
}

// BranchCompletions returns the remote branches matching the typed prefix. An
// empty prefix returns everything, so Tab on an empty field cycles through all
// branches.
func BranchCompletions(branches []string, prefix string) []string {
	prefix = strings.TrimSpace(prefix)
	matches := make([]string, 0, len(branches))
	for _, branch := range branches {
		if prefix == "" || strings.HasPrefix(branch, prefix) {
			matches = append(matches, branch)
		}
	}
	return matches
}

// Pull runs `git pull --ff-only` inside an existing clone. A fast-forward-only
// strategy is deliberate: GoFetch never creates merge commits or rebases in a
// repository it does not own, it only catches the clone up with its remote
// when that is trivially safe.
func Pull(ctx context.Context, target string) (summary string, err error) {
	info, statErr := os.Stat(target)
	if statErr != nil || !info.IsDir() {
		return "", fmt.Errorf("la cartella non esiste più: %s", target)
	}
	cmd := GitCommand(ctx, "-C", target, "pull", "--ff-only")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	runErr := cmd.Run()
	text := strings.TrimSpace(output.String())
	if runErr != nil {
		return "", FriendlyPullError(runErr, text)
	}
	return SummarizePull(text), nil
}

// FriendlyPullError condenses git's output into one actionable sentence.
func FriendlyPullError(err error, output string) error {
	if matched := friendlyError(err, output, pullRules); matched != nil {
		return matched
	}
	return fmt.Errorf("git pull non riuscito: %s", CompactError(err))
}

var pullRules = []friendlyRule{
	{keywords: []string{"not a git repository"}, build: func() error {
		return errText("la cartella non è un repository Git")
	}},
	{keywords: []string{"not possible to fast-forward", "divergent"}, build: func() error {
		return errText("aggiornamento non automatico: il branch ha modifiche divergenti, risolvile con git")
	}},
	{keywords: []string{"local changes", "would be overwritten"}, build: func() error {
		return errText("ci sono modifiche locali: fai commit o stash prima di aggiornare")
	}},
	authRule,
	networkRule,
}

// SummarizePull turns git's success output into a short status line.
func SummarizePull(output string) string {
	lower := strings.ToLower(output)
	switch {
	case output == "":
		return "Repository aggiornato."
	case strings.Contains(lower, "already up to date") || strings.Contains(lower, "already up-to-date"):
		return "Già aggiornato: nessuna novità dal remote."
	default:
		// The "Updating <from>..<to>" line is the most informative part of a
		// fast-forward: it says exactly how far the clone moved.
		for _, line := range strings.Split(output, "\n") {
			if trimmed := strings.TrimSpace(line); strings.HasPrefix(strings.ToLower(trimmed), "updating ") {
				return "Aggiornato: " + trimmed
			}
		}
		if line := LastMeaningfulLine(output); line != "" {
			return "Aggiornato: " + line
		}
		return "Repository aggiornato."
	}
}

// OpenFolder asks the platform file manager to show target. Start (rather
// than Run) is used on purpose: the opener detaches immediately so the UI
// never waits on the file manager closing.
func OpenFolder(target string) error {
	target = ExpandHome(target)
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}

// CompactError squeezes an error into a single short line for status bars.
func CompactError(err error) string {
	message := strings.TrimSpace(err.Error())
	if len(message) > 160 {
		return message[:157] + "..."
	}
	return message
}

// LastMeaningfulLine returns the final non-empty line of output — usually the
// actual error in a tool's log.
func LastMeaningfulLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		if line := strings.TrimSpace(lines[index]); line != "" {
			return line
		}
	}
	return ""
}

// DefaultGitTimeout bounds background git operations such as ls-remote.
const DefaultGitTimeout = 15 * time.Second
