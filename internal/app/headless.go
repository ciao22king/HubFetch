package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ciao22king/GoFetch/internal/core"
)

// RunHeadless clones a repository without the TUI: it is what powers
// `gofetch -url <url> -yes`, so the tool can be used inside scripts and CI
// pipelines where no terminal interaction is possible. Progress goes to
// stderr and the final destination path to stdout, so the path can be
// captured with command substitution.
func RunHeadless(opts Options) error {
	repo, err := core.ParseRepository(opts.InitialURL)
	if err != nil {
		return err
	}

	parent := opts.InitialDir
	if parent == "" {
		var wdErr error
		if parent, wdErr = os.Getwd(); wdErr != nil {
			return wdErr
		}
	}

	target, err := core.DestinationPath(parent, repo)
	if err != nil {
		return err
	}
	if core.NonEmptyDir(target) {
		if !opts.Overwrite {
			return fmt.Errorf("la cartella %s esiste già e non è vuota (usa -force per sostituirla)", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	branch := strings.TrimSpace(opts.InitialBranch)
	fmt.Fprintf(os.Stderr, "Clono %s (%s) in %s…\n", repo.Name, repo.Provider, target)

	started := time.Now()
	if err := headlessClone(repo, branch, opts.Depth, target); err != nil {
		return err
	}
	elapsed := time.Since(started).Round(time.Second)

	if !opts.NoBuild {
		// The history write runs concurrently with the build: one disk round-trip
		// hidden behind several seconds of compilation.
		var historyWG sync.WaitGroup
		historyWG.Add(1)
		go func() {
			defer historyWG.Done()
			saveHistoryAsync(core.Entry{Repository: repo, Target: target})
		}()

		build := core.BuildProject(context.Background(), target)
		switch {
		case build.Skipped:
			fmt.Fprintln(os.Stderr, "Nessuna build riconosciuta: clone completato.")
		case build.Err != nil:
			fmt.Fprintf(os.Stderr, "Attenzione: build fallita (%s): %s\n", build.Command, core.CompactError(build.Err))
		default:
			fmt.Fprintf(os.Stderr, "Build completata (%s) in %s.\n", build.Command, build.Duration.Round(time.Second))
		}
		historyWG.Wait()
	} else {
		saveHistoryAsync(core.Entry{Repository: repo, Target: target})
	}

	fmt.Fprintf(os.Stderr, "Fatto in %s.\n", elapsed)
	fmt.Println(target)
	return nil
}

// saveHistoryAsync upserts an entry into the on-disk history without ever
// failing the fetch: a broken history file must not undo a successful clone.
func saveHistoryAsync(entry core.Entry) {
	dir, err := core.DefaultStoreDir()
	if err != nil {
		return
	}
	store := core.NewStore(dir)
	_ = store.Save(core.Upsert(store.Load(), entry))
}

// headlessClone runs git clone with progress streamed straight to stderr. The
// scanner goroutine writes output lines concurrently with git, so cloning a
// large repository never stalls on a synchronous print.
func headlessClone(repo core.Repository, branch string, depth int, target string) error {
	args := []string{"clone", "--progress"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	args = append(args, repo.URL, target)

	cmd := core.GitCommand(context.Background(), args...)

	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	var combined strings.Builder
	var combinedMu sync.Mutex
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			combinedMu.Lock()
			combined.WriteString(line)
			combined.WriteByte('\n')
			combinedMu.Unlock()
			fmt.Fprintln(os.Stderr, line)
		}
	}()

	runErr := cmd.Run()
	writer.Close()
	<-done

	combinedMu.Lock()
	output := combined.String()
	combinedMu.Unlock()
	return core.FriendlyCloneError(runErr, output, target)
}
