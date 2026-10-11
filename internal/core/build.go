package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BuildTimeout bounds any automatic build so a hung toolchain can never wedge
// a fetch forever.
const BuildTimeout = 10 * time.Minute

// BuildPlan is the command that builds a detected project type.
type BuildPlan struct {
	Command string
	Args    []string
	Label   string // human-readable form shown in status lines
}

// BuildResult reports how a build went.
type BuildResult struct {
	Command  string
	Output   string
	Err      error
	Duration time.Duration
	Skipped  bool
}

// Manifests checked by DetectBuildPlan, in priority order. Declared once at
// package level so detection does not rebuild the list on every call.
var manifestChecks = []struct {
	files []string
	plan  BuildPlan
}{
	{[]string{"go.mod"}, BuildPlan{Command: "go", Args: []string{"build", "./..."}, Label: "go build ./..."}},
	{[]string{"Cargo.toml"}, BuildPlan{Command: "cargo", Args: []string{"build"}, Label: "cargo build"}},
	{[]string{"pom.xml"}, BuildPlan{Command: "mvn", Args: []string{"-q", "-DskipTests", "package"}, Label: "mvn package"}},
	{[]string{"build.gradle", "build.gradle.kts"}, BuildPlan{Command: "gradle", Args: []string{"build"}, Label: "gradle build"}},
	{[]string{"Makefile", "makefile"}, BuildPlan{Command: "make", Label: "make"}},
	{[]string{"pyproject.toml"}, BuildPlan{Command: "python3", Args: []string{"-m", "build"}, Label: "python -m build"}},
	{[]string{"setup.py"}, BuildPlan{Command: "python3", Args: []string{"setup.py", "build"}, Label: "python setup.py build"}},
}

// DetectBuildPlan looks for a recognizable project manifest and returns the
// command that builds it. It intentionally avoids installing dependencies:
// only real build commands run, and only for repositories you chose to clone.
// All top-level manifests are stat'ed concurrently, which keeps detection to
// roughly one disk round-trip even on slow filesystems.
func DetectBuildPlan(target string) BuildPlan {
	type outcome struct {
		index int
		found bool
	}
	results := make(chan outcome, len(manifestChecks))
	var wg sync.WaitGroup
	for index, check := range manifestChecks {
		wg.Add(1)
		go func(index int, files []string) {
			defer wg.Done()
			for _, name := range files {
				if exists(filepath.Join(target, name)) {
					results <- outcome{index: index, found: true}
					return
				}
			}
			results <- outcome{index: index}
		}(index, check.files)
	}
	wg.Wait()
	close(results)

	best := -1
	for result := range results {
		if result.found && (best < 0 || result.index < best) {
			best = result.index
		}
	}
	if best >= 0 {
		return manifestChecks[best].plan
	}
	return nodeBuildPlan(target)
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

// nodeBuildPlan inspects package.json for a build-ish script and picks the
// package manager indicated by the lockfile.
func nodeBuildPlan(target string) BuildPlan {
	data, err := os.ReadFile(filepath.Join(target, "package.json"))
	if err != nil {
		return BuildPlan{}
	}
	var manifest packageJSON
	if json.Unmarshal(data, &manifest) != nil {
		return BuildPlan{}
	}

	script := firstScript(manifest.Scripts, "build", "compile", "dist", "bundle")
	if script == "" {
		return BuildPlan{}
	}

	switch detectNodeManager(target) {
	case "yarn":
		return BuildPlan{Command: "yarn", Args: []string{script}, Label: "yarn " + script}
	case "pnpm":
		return BuildPlan{Command: "pnpm", Args: []string{"run", script}, Label: "pnpm run " + script}
	default:
		return BuildPlan{Command: "npm", Args: []string{"run", script}, Label: "npm run " + script}
	}
}

func firstScript(scripts map[string]string, names ...string) string {
	for _, name := range names {
		if strings.TrimSpace(scripts[name]) != "" {
			return name
		}
	}
	return ""
}

func detectNodeManager(target string) string {
	switch {
	case exists(filepath.Join(target, "pnpm-lock.yaml")):
		return "pnpm"
	case exists(filepath.Join(target, "yarn.lock")):
		return "yarn"
	default:
		return "npm"
	}
}

// BuildProject runs the detected build for target, honouring BuildTimeout. If
// no supported setup is found the result is simply skipped — the clone still
// completes normally.
func BuildProject(ctx context.Context, target string) BuildResult {
	plan := DetectBuildPlan(target)
	if plan.Command == "" {
		return BuildResult{Skipped: true}
	}

	started := time.Now()
	command := resolveCommand(plan.Command)
	if command == "" {
		return BuildResult{
			Command:  plan.Label,
			Err:      fmt.Errorf("%s non trovato nel PATH", plan.Command),
			Duration: time.Since(started),
		}
	}

	ctx, cancel := context.WithTimeout(ctx, BuildTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, plan.Args...)
	cmd.Dir = target
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		err = fmt.Errorf("timeout dopo %d minuti", int(BuildTimeout.Minutes()))
	} else if err != nil {
		if detail := LastMeaningfulLine(output.String()); detail != "" {
			err = fmt.Errorf("%v — %s", err, detail)
		}
	}

	return BuildResult{
		Command:  plan.Label,
		Output:   strings.TrimSpace(output.String()),
		Err:      err,
		Duration: time.Since(started),
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolveCommand returns the executable to run, or "" if none is available.
// Python is called python3 on macOS/Linux and python on Windows, so fall back
// to whichever exists.
func resolveCommand(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	if name == "python3" {
		if path, err := exec.LookPath("python"); err == nil {
			return path
		}
	}
	return ""
}
