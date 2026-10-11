package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ciao22king/GoFetch/internal/core"
)

// writePopulatedDir creates dir with one file inside, so NonEmptyDir sees it.
func writePopulatedDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "existing"), []byte("x"), 0o644)
}

func newTestModel(t *testing.T, opts Options) model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AppData", t.TempDir()) // Windows config location
	if opts.Version == "" {
		opts.Version = "test"
	}
	return newModel(opts)
}

func apply(t *testing.T, m model, msg tea.Msg) (model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	next, ok := updated.(model)
	if !ok {
		t.Fatalf("Update() returned %T, want model", updated)
	}
	return next, cmd
}

func TestFlowURLToBranchToDestination(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	if m.screen != urlScreen {
		t.Fatalf("initial screen = %v, want urlScreen", m.screen)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != branchScreen {
		t.Fatalf("screen after URL = %v, want branchScreen", m.screen)
	}
	if m.repository.Name != "project" {
		t.Fatalf("parsed repository name = %q, want project", m.repository.Name)
	}

	m.branchInput.SetValue("develop")
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != destinationScreen {
		t.Fatalf("screen after branch = %v, want destinationScreen", m.screen)
	}
}

func TestBranchStepCanBeLeftEmpty(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m.dirInput.SetValue(t.TempDir())
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != cloningScreen {
		t.Fatalf("screen = %v, want cloningScreen", m.screen)
	}
	if m.cloneBranch != "" {
		t.Fatalf("cloneBranch = %q, want empty for the default branch", m.cloneBranch)
	}
	if m.cancelClone != nil {
		defer m.cancelClone() // don't leak the pending clone goroutine
	}
}

func TestBranchStepCapturesBranch(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.branchInput.SetValue("release/1.x")
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m.dirInput.SetValue(t.TempDir())
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != cloningScreen {
		t.Fatalf("screen = %v, want cloningScreen", m.screen)
	}
	if m.cloneBranch != "release/1.x" {
		t.Fatalf("cloneBranch = %q, want release/1.x", m.cloneBranch)
	}
	if m.cancelClone != nil {
		defer m.cancelClone()
	}
}

func TestBranchRejectsSpaces(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.branchInput.SetValue("bad branch")
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != branchScreen {
		t.Fatalf("screen = %v, want to stay on branchScreen for an invalid branch", m.screen)
	}
	if m.err == nil {
		t.Fatal("expected a validation error for a branch containing spaces")
	}
}

func TestBranchEscReturnsToURL(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.screen != urlScreen {
		t.Fatalf("screen = %v, want urlScreen after esc on the branch step", m.screen)
	}
}

func TestProgressLinesAreCapturedAndBounded(t *testing.T) {
	m := newTestModel(t, Options{})
	m.screen = cloningScreen
	for i := 0; i < maxProgressLines+50; i++ {
		m, _ = apply(t, m, progressMsg{line: "Receiving objects: 42%"})
	}
	if len(m.progressLog) != maxProgressLines {
		t.Fatalf("progressLog length = %d, want %d", len(m.progressLog), maxProgressLines)
	}

	// Blank lines are ignored so the view does not fill with git's carriage
	// return padding.
	before := len(m.progressLog)
	m, _ = apply(t, m, progressMsg{line: "   "})
	if len(m.progressLog) != before {
		t.Fatal("blank progress line should be ignored")
	}
}

func TestTailLinesAndTruncateHelpers(t *testing.T) {
	values := []string{"a", "b", "c", "d"}
	if got := tailLines(values, 2); len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Fatalf("tailLines() = %v, want [c d]", got)
	}
	if got := tailLines(values, 10); len(got) != 4 {
		t.Fatalf("tailLines() = %v, want all values", got)
	}
	if got := truncateLine("  short  ", 10); got != "short" {
		t.Fatalf("truncateLine() = %q, want short", got)
	}
	if got := truncateLine("abcdefghij", 5); len([]rune(got)) != 5 {
		t.Fatalf("truncateLine() = %q, want 5 runes", got)
	}
}

func TestInitialBranchSkipsBranchScreen(t *testing.T) {
	m := newTestModel(t, Options{
		InitialURL:    "https://github.com/owner/project",
		InitialBranch: "develop",
	})
	if m.screen != destinationScreen {
		t.Fatalf("screen = %v, want destinationScreen when URL and branch are given", m.screen)
	}
	if m.branchInput.Value() != "develop" {
		t.Fatalf("branchInput = %q, want develop", m.branchInput.Value())
	}
	if m.repository.Name != "project" {
		t.Fatalf("repository name = %q, want project", m.repository.Name)
	}
}

func TestViewsDoNotPanicOnEveryScreen(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m.width, m.height = 100, 40
	m.repository = core.Repository{Provider: "GitHub", Name: "project", URL: "https://github.com/owner/project"}

	for _, screen := range []screen{historyScreen, urlScreen, branchScreen, destinationScreen, cloningScreen, resultScreen} {
		m.screen = screen
		if view := m.View(); view == "" {
			t.Fatalf("View() for screen %v is empty", screen)
		}
	}
}

func TestInvalidURLKeepsUserOnURLScreen(t *testing.T) {
	m := newTestModel(t, Options{})
	m.screen = urlScreen
	m.urlInput.SetValue("https://github.com/owner")
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != urlScreen {
		t.Fatalf("screen = %v, want urlScreen after invalid URL", m.screen)
	}
	if m.err == nil {
		t.Fatal("expected a validation error for an incomplete URL")
	}
}

func TestQuitKeyIgnoredWhileTyping(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, cmd := apply(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil || m.quitting {
		t.Fatal("pressing q while typing a URL must not quit")
	}
}

func TestOverwriteConfirmation(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, "project")
	if err := writePopulatedDir(target); err != nil {
		t.Fatal(err)
	}

	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.dirInput.SetValue(parent)

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.confirmOverwrite {
		t.Fatal("expected an overwrite confirmation for a non-empty destination")
	}
	if m.screen != destinationScreen {
		t.Fatalf("screen = %v, want destinationScreen while confirming", m.screen)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.confirmOverwrite {
		t.Fatal("Esc must cancel the overwrite confirmation")
	}
}

func TestEscCancelsClone(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.dirInput.SetValue(t.TempDir())

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.screen != cloningScreen {
		t.Fatalf("screen = %v, want cloningScreen", m.screen)
	}
	if m.cancelClone == nil {
		t.Fatal("expected a cancel function to be stored while cloning")
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.cancelClone == nil {
		t.Fatal("cancel function should remain until the clone result arrives")
	}
	m.cancelClone() // don't leak the pending clone goroutine
}

func TestFetchSuccessStoresHistory(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.cancelClone != nil {
		defer m.cancelClone()
	}

	m, _ = apply(t, m, fetchResultMsg{target: filepath.Join(t.TempDir(), "project")})
	if m.screen != historyScreen {
		t.Fatalf("screen = %v, want historyScreen after success", m.screen)
	}
	if len(m.history) != 1 {
		t.Fatalf("history length = %d, want 1", len(m.history))
	}
}

func TestFetchFailureShowsResultScreen(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.cancelClone != nil {
		defer m.cancelClone()
	}

	m, _ = apply(t, m, fetchResultMsg{err: core.ErrCloneCanceled})
	if m.screen != resultScreen {
		t.Fatalf("screen = %v, want resultScreen after failure", m.screen)
	}
}

func TestHistoryNavigationAndSearch(t *testing.T) {
	m := newTestModel(t, Options{})
	m.history = []core.Entry{
		{Repository: core.Repository{Name: "alpha", Provider: "GitHub"}, Target: "/code/alpha"},
		{Repository: core.Repository{Name: "beta", Provider: "GitLab"}, Target: "/work/beta"},
	}
	m.refreshFilter()

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.selected != 1 {
		t.Fatalf("selected = %d, want 1 after down", m.selected)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyCtrlL})
	if !m.searchActive {
		t.Fatal("ctrl+l should activate the search field")
	}
	for _, r := range "beta" {
		m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(m.filtered) != 1 || m.history[m.filtered[0]].Name != "beta" {
		t.Fatalf("filtered = %v, want only beta", m.filtered)
	}

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchActive {
		t.Fatal("Esc should close the search field")
	}
}

func TestHistoryRemoveEntry(t *testing.T) {
	m := newTestModel(t, Options{})
	m.history = []core.Entry{
		{Repository: core.Repository{Name: "alpha"}, Target: "/code/alpha"},
	}
	m.refreshFilter()
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyCtrlD})
	if len(m.history) != 0 {
		t.Fatalf("history length = %d, want 0 after ctrl+d", len(m.history))
	}
}

func TestTabCyclesDestinationSuggestions(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m.history = []core.Entry{
		{Target: filepath.Join("/code", "alpha")},
		{Target: filepath.Join("/work", "beta")},
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyTab})
	first := m.dirInput.Value()
	if first == "" {
		t.Fatal("tab did not produce a destination suggestion")
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.dirInput.Value() == first {
		t.Fatal("tab did not cycle to a different suggestion")
	}
}

func TestShallowToggleSwitchesDepth(t *testing.T) {
	m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.depth != 1 {
		t.Fatalf("depth = %d, want 1 after enabling the shallow toggle", m.depth)
	}
	m, _ = apply(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if m.depth != 0 {
		t.Fatalf("depth = %d, want 0 after disabling the shallow toggle", m.depth)
	}
}
