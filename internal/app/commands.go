package app

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ciao22king/GoFetch/internal/core"
	"github.com/ciao22king/GoFetch/internal/fetch"
)

// fetchResultMsg reports the outcome of a clone (+build) run. It keeps the
// same field names as before so the result view and tests stay expressive.
type fetchResultMsg struct {
	target   string
	output   string
	err      error
	duration time.Duration
	build    core.BuildResult
}

type progressMsg struct{ line string }

// progressTickMsg is delivered on a fixed interval while a clone is running so
// the elapsed timer in the cloning view keeps counting even when git is quiet.
type progressTickMsg struct{}

const cloneProgressInterval = time.Second

type clipboardMsg struct {
	value string
	err   error
}

type copiedMsg struct{ err error }

type openResultMsg struct {
	target string
	err    error
}

// branchesMsg delivers the branch names advertised by the remote, fetched in
// the background while the user is on the branch screen. The URL is echoed
// back so a stale response can be discarded.
type branchesMsg struct {
	url      string
	branches []string
	err      error
}

// pullResultMsg reports the outcome of updating an existing clone from the
// history screen.
type pullResultMsg struct {
	target   string
	summary  string
	err      error
	duration time.Duration
}

func readClipboard() tea.Cmd {
	return func() tea.Msg {
		value, err := core.ReadClipboard()
		return clipboardMsg{value: value, err: err}
	}
}

func openRepository(target string) tea.Cmd {
	return func() tea.Msg {
		err := core.OpenFolder(target)
		return openResultMsg{target: target, err: err}
	}
}

// fetchRemoteBranches lists the remote's branches without cloning anything.
func fetchRemoteBranches(repoURL string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), core.DefaultGitTimeout)
		defer cancel()
		branches, err := core.RemoteBranches(ctx, repoURL)
		return branchesMsg{url: repoURL, branches: branches, err: err}
	}
}

// pullRepository runs `git pull --ff-only` inside an existing clone.
func pullRepository(target string) tea.Cmd {
	return func() tea.Msg {
		started := time.Now()
		summary, err := core.Pull(context.Background(), core.ExpandHome(target))
		return pullResultMsg{target: target, summary: summary, err: err, duration: time.Since(started)}
	}
}

// startFetch launches the full clone+build pipeline in a cancellable
// goroutine. Progress lines flow through the buffered channel so a slow UI
// render can never block git; waitForProgress re-arms the reader after each
// line. The returned CancelFunc aborts a stuck network fetch (Esc).
func startFetch(repo core.Repository, parent, branch string, overwrite, noBuild bool, depth int, store *core.Store, progress chan<- string) (tea.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := func() tea.Msg {
		var lineBuffer = make(chan string, 256)
		done := make(chan struct{})
		go func() {
			defer close(done)
			for line := range lineBuffer {
				select {
				case progress <- line:
				default: // drop rather than block when the UI falls behind
				}
			}
		}()

		result, err := fetch.Run(ctx, fetch.Options{
			RawURL:    repo.URL,
			Branch:    branch,
			Parent:    parent,
			Depth:     depth,
			Overwrite: overwrite,
			NoBuild:   noBuild,
			Progress: func(line string) {
				lineBuffer <- line
			},
		}, storeProxy{store})

		close(lineBuffer)
		<-done
		if result == nil {
			return fetchResultMsg{err: err}
		}
		return fetchResultMsg{
			target:   result.Target,
			output:   result.Output,
			err:      err,
			duration: result.Duration,
			build:    result.Build,
		}
	}
	return cmd, cancel
}

// storeProxy adapts *core.Store to fetch.Store, tolerating a nil store (the
// fetch then skips history persistence).
type storeProxy struct{ store *core.Store }

func (p storeProxy) Load() []core.Entry {
	if p.store == nil {
		return nil
	}
	return p.store.Load()
}

func (p storeProxy) Save(entries []core.Entry) error {
	if p.store == nil {
		return nil
	}
	return p.store.Save(entries)
}

// progressTick schedules the next one-second timer for the cloning view.
func progressTick() tea.Cmd {
	return tea.Tick(cloneProgressInterval, func(time.Time) tea.Msg { return progressTickMsg{} })
}

// waitForProgress blocks on the progress channel until git emits a line or the
// channel is closed, then hands the line to the TUI. The model re-arms it
// after every line so progress keeps flowing.
func waitForProgress(ch chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return progressMsg{line: line}
	}
}
