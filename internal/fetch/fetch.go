// Package fetch drives a complete GoFetch operation — resolve destination,
// optional overwrite, clone with live progress, optional build, record
// history — for both the interactive TUI and the headless (--yes) mode, so
// the two can never drift apart.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ciao22king/GoFetch/internal/core"
)

// Options describes one fetch request.
type Options struct {
	RawURL    string // repository URL as pasted by the user
	Branch    string // empty = remote default branch
	Parent    string // parent directory; empty = current working directory
	Depth     int    // > 0 = shallow clone
	Overwrite bool   // replace an existing non-empty destination
	NoBuild   bool   // skip the automatic build
	Progress  func(line string)
}

// Result is what a finished fetch produced.
type Result struct {
	Repo     core.Repository
	Target   string
	Output   string
	Duration time.Duration
	Build    core.BuildResult
}

// Store persists the history; implemented by *core.Store.
type Store interface {
	Load() []core.Entry
	Save(entries []core.Entry) error
}

// Run performs the whole fetch synchronously: it is safe to call from a
// goroutine and honours ctx cancellation (a cancelled clone cleans up the
// half-written directory). The history is updated only on success; failing to
// persist it never fails the fetch itself.
func Run(ctx context.Context, opts Options, store Store) (*Result, error) {
	repo, err := core.ParseRepository(opts.RawURL)
	if err != nil {
		return nil, err
	}

	parent := opts.Parent
	if parent == "" {
		if parent, err = os.Getwd(); err != nil {
			return nil, err
		}
	}
	target, err := core.DestinationPath(parent, repo)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	if core.NonEmptyDir(target) {
		if !opts.Overwrite {
			return nil, fmt.Errorf("la cartella %s esiste già e non è vuota", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}

	output, err := core.Clone(ctx, core.CloneOptions{
		Repo:     repo,
		Target:   target,
		Branch:   opts.Branch,
		Depth:    opts.Depth,
		Progress: opts.Progress,
	})
	if err != nil {
		if errors.Is(err, core.ErrCloneCanceled) {
			// git leaves a half-written directory behind on cancel; remove it
			// so the next attempt starts from a clean state.
			_ = os.RemoveAll(target)
		}
		return nil, err
	}

	result := &Result{
		Repo:     repo,
		Target:   target,
		Output:   output,
		Duration: time.Since(started),
		Build:    core.BuildResult{Skipped: true},
	}
	if !opts.NoBuild {
		result.Build = core.BuildProject(ctx, target)
	}

	if store != nil {
		entry := core.Entry{Repository: repo, Target: target}
		_ = store.Save(core.Upsert(store.Load(), entry))
	}
	return result, nil
}
