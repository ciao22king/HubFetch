// Package tui implements GoFetch's Bubble Tea interface. It is deliberately
// thin: every piece of logic lives in internal/core and internal/fetch, so
// this package only translates messages into screens and back.
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ciao22king/GoFetch/internal/core"
)

type screen int

const (
	historyScreen screen = iota
	urlScreen
	branchScreen
	destinationScreen
	cloningScreen
	resultScreen
)

// Options configures a GoFetch interactive run.
type Options struct {
	InitialURL    string
	InitialBranch string
	InitialDir    string
	Version       string
	NoBuild       bool
	// Depth limits the clone to that many commits (0 = full clone).
	Depth int
	// Overwrite replaces an existing non-empty destination without asking.
	// Only meaningful for headless runs; the TUI always asks first.
	Overwrite bool
}

// maxProgressLines bounds how much clone output the UI keeps in memory and
// shows; older lines scroll out of view.
const maxProgressLines = 200

type model struct {
	screen      screen
	urlInput    textinput.Model
	branchInput textinput.Model
	dirInput    textinput.Model
	searchInput textinput.Model
	spinner     spinner.Model
	repository  core.Repository
	result      fetchResultMsg
	history     []core.Entry
	filtered    []int
	selected    int
	err         error
	status      string
	width       int
	height      int
	version     string
	startedAt   time.Time
	quitting    bool
	// urlSubmitted blocks duplicate Enter presses on the URL screen from
	// launching parallel ls-remote lookups for the same repository.
	urlSubmitted     bool
	searchActive     bool
	confirmOverwrite bool
	suggestionIndex  int
	cancelClone      context.CancelFunc
	noBuild          bool

	progressChan chan string
	progressLog  []string
	cloneBranch  string

	// depth is the shallow-clone depth chosen for the next clone (0 = full).
	depth int
	// remoteBranches caches the branch names advertised by the current
	// repository's remote; branchCycling tracks Tab-completion position.
	remoteBranches []string
	branchCycling  int
	// pendingPull is the target currently being updated with git pull, so the
	// history screen can show that work is in progress.
	pendingPull string

	store *core.Store
}

var (
	primary = lipgloss.Color("#8B5CF6")
	cyan    = lipgloss.Color("#22D3EE")
	green   = lipgloss.Color("#34D399")
	muted   = lipgloss.Color("#94A3B8")
	ink     = lipgloss.Color("#E2E8F0")
	danger  = lipgloss.Color("#FB7185")

	// Styles are built once at startup: rebuilding them per frame was the
	// single biggest cost of the old View().
	logoStyle = lipgloss.NewStyle().
			Foreground(ink).
			Bold(true).
			Background(primary).
			Padding(0, 1)
	subtitleStyle = lipgloss.NewStyle().Foreground(muted)
	labelStyle    = lipgloss.NewStyle().Foreground(muted).Bold(true)
	cardStyle     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#334155")).
			Padding(1, 2)
	helpStyle     = lipgloss.NewStyle().Foreground(muted)
	titleStyle    = lipgloss.NewStyle().Foreground(ink).Bold(true)
	badgeStyle    = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	rowNameStyle  = lipgloss.NewStyle().Foreground(ink)
	rowMetaStyle  = lipgloss.NewStyle().Foreground(muted)
	errorStyle    = lipgloss.NewStyle().Foreground(danger)
	dangerStyle   = lipgloss.NewStyle().Foreground(danger).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(green).Bold(true)
	targetStyle   = lipgloss.NewStyle().Foreground(ink)
	progressStyle = lipgloss.NewStyle().Foreground(muted)
	pillStyle     = lipgloss.NewStyle().Foreground(muted).Background(lipgloss.Color("#1E293B")).Padding(0, 1)
	shallowPill   = lipgloss.NewStyle().Foreground(cyan).Background(lipgloss.Color("#1E293B")).Padding(0, 1)
	frameStyle    = lipgloss.NewStyle() // width/margins set per resize in newModel/WindowSize
)

type keys struct {
	quit   key.Binding
	enter  key.Binding
	back   key.Binding
	open   key.Binding
	remove key.Binding
	copy   key.Binding
	search key.Binding
}

var appKeys = keys{
	quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
	enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	),
	back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	open: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open folder"),
	),
	remove: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "remove"),
	),
	copy: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy path"),
	),
	search: key.NewBinding(
		key.WithKeys("ctrl+l", "/"),
		key.WithHelp("ctrl+l", "search"),
	),
}

// Run starts the interactive program.
func Run(opts Options) error {
	m := newModel(opts)
	program := tea.NewProgram(&m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(opts Options) model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://github.com/owner/repository"
	urlInput.Prompt = "  "
	urlInput.CharLimit = 500
	urlInput.Width = 60
	urlInput.SetValue(opts.InitialURL)

	branchInput := textinput.New()
	branchInput.Placeholder = "main (vuoto = branch predefinito)"
	branchInput.Prompt = "  "
	branchInput.CharLimit = 200
	branchInput.Width = 60

	dirInput := textinput.New()
	dirInput.Placeholder = "~/Code"
	dirInput.Prompt = "  "
	dirInput.CharLimit = 500
	dirInput.Width = 60
	dirInput.SetValue(opts.InitialDir)

	searchInput := textinput.New()
	searchInput.Placeholder = "cerca per nome, provider o percorso"
	searchInput.Prompt = "  / "
	searchInput.CharLimit = 200
	searchInput.Width = 60

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(cyan)

	storeDir, storeErr := core.DefaultStoreDir()
	store := core.NewStore(storeDir)

	m := model{
		urlInput:      urlInput,
		branchInput:   branchInput,
		dirInput:      dirInput,
		searchInput:   searchInput,
		spinner:       spin,
		version:       opts.Version,
		history:       store.Load(),
		noBuild:       opts.NoBuild,
		depth:         opts.Depth,
		branchCycling: -1,
		store:         store,
	}
	if storeErr != nil {
		m.store = nil // nowhere to persist: keep running without history
		m.history = nil
	}
	m.refreshFilter()

	// The URL is parsed exactly once up front; a bad one only shows an error.
	parsedRepo, parseErr := core.ParseRepository(opts.InitialURL)
	urlIsValid := opts.InitialURL != "" && parseErr == nil

	switch {
	case urlIsValid && opts.InitialBranch != "":
		// Both provided on the command line: skip straight to the destination.
		m.repository = parsedRepo
		m.branchInput.SetValue(opts.InitialBranch)
		m.screen = destinationScreen
		m.dirInput.Focus()
	case urlIsValid:
		m.screen = urlScreen
		m.urlInput.Focus()
	default:
		m.screen = historyScreen
		if opts.InitialURL != "" && parseErr != nil {
			// Invalid URL from the command line: show it so the user can fix it.
			m.err = parseErr
			m.urlInput.SetValue(opts.InitialURL)
			m.urlInput.Focus()
			m.screen = urlScreen
		}
	}
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// refreshFilter recomputes the visible history indexes and keeps the selection
// inside bounds. It is called whenever the history or the search query changes.
func (m *model) refreshFilter() {
	m.filtered = core.Search(m.history, m.searchInput.Value())
	if len(m.filtered) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = len(m.filtered) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		inputWidth := min(70, max(35, msg.Width-16))
		m.urlInput.Width = inputWidth
		m.branchInput.Width = inputWidth
		m.dirInput.Width = inputWidth
		m.searchInput.Width = inputWidth
		return m, nil
	case clipboardMsg:
		return m.applyClipboard(msg)
	case copiedMsg:
		if msg.err != nil {
			m.status = "Copia non riuscita: " + core.CompactError(msg.err)
		} else {
			m.status = "Percorso copiato nella clipboard."
		}
		return m, nil
	case progressMsg:
		return m.recordProgress(msg)
	case progressTickMsg:
		if m.screen == cloningScreen {
			return m, progressTick()
		}
		return m, nil
	case fetchResultMsg:
		return m.applyFetchResult(msg)
	case openResultMsg:
		if msg.err != nil {
			m.status = "Non riesco ad aprire la cartella: " + core.CompactError(msg.err)
		} else {
			m.status = "Cartella aperta: " + msg.target
		}
		return m, nil
	case branchesMsg:
		// Discard stale answers: the user may have gone back and picked a
		// different repository while ls-remote was still running.
		if msg.err == nil && msg.url == m.repository.URL {
			m.remoteBranches = msg.branches
		}
		return m, nil
	case pullResultMsg:
		m.pendingPull = ""
		if msg.err != nil {
			m.status = "Aggiornamento fallito: " + core.CompactError(msg.err)
		} else {
			m.status = msg.summary
		}
		return m, nil
	case tea.KeyMsg:
		// Global shortcuts (ctrl+c, open, copy, quit…) win over the screens —
		// but only when they actually *consume* the key. handleGlobalKey returns
		// handled=true exclusively for keys it acted upon; plain typing like "q"
		// inside a focused text field is deliberately left unclaimed so it
		// reaches the input below and can never quit the app while typing.
		if cmd, handled := m.handleGlobalKey(msg); handled {
			return m, cmd
		}
	}

	switch m.screen {
	case historyScreen:
		return m.updateHistory(msg)
	case urlScreen:
		return m.updateURL(msg)
	case branchScreen:
		return m.updateBranch(msg)
	case destinationScreen:
		return m.updateDestination(msg)
	case cloningScreen:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && key.Matches(keyMsg, appKeys.back) {
			if m.cancelClone != nil {
				m.cancelClone()
				m.status = "Annullamento del clone…"
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case resultScreen:
		return m.updateResult(msg)
	}
	return m, nil
}

func (m *model) applyClipboard(msg clipboardMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = "Clipboard non disponibile: " + core.CompactError(msg.err)
		return m, nil
	}
	value := strings.TrimSpace(msg.value)
	if value == "" {
		m.status = "La clipboard è vuota."
		return m, nil
	}
	switch m.screen {
	case urlScreen:
		m.urlInput.SetValue(value)
		m.status = "URL incollato dalla clipboard."
	case branchScreen:
		m.branchInput.SetValue(value)
		m.status = "Branch incollato dalla clipboard."
	case destinationScreen:
		m.dirInput.SetValue(value)
		m.status = "Percorso incollato dalla clipboard."
	}
	return m, nil
}

func (m *model) recordProgress(msg progressMsg) (tea.Model, tea.Cmd) {
	if m.screen == cloningScreen && strings.TrimSpace(msg.line) != "" {
		m.progressLog = append(m.progressLog, msg.line)
		if len(m.progressLog) > maxProgressLines {
			m.progressLog = m.progressLog[len(m.progressLog)-maxProgressLines:]
		}
	}
	if m.progressChan != nil {
		return m, waitForProgress(m.progressChan)
	}
	return m, nil
}

func (m *model) applyFetchResult(msg fetchResultMsg) (tea.Model, tea.Cmd) {
	m.cancelClone = nil
	m.result = msg
	m.progressChan = nil
	if msg.err == nil {
		entry := core.Entry{Repository: m.repository, Target: msg.target}
		if m.store != nil {
			m.history = core.Upsert(m.history, entry)
			if err := m.store.Save(m.history); err != nil {
				m.status = "Clone completato, ma non riesco a salvare la cronologia."
			}
		}
		m.selected = 0
		m.refreshFilter()
		switch {
		case m.store == nil:
			m.status = "Clone completato."
		case msg.build.Command != "" && msg.build.Err != nil:
			m.status = fmt.Sprintf("Clone completato, ma la build è fallita (%s): %s", msg.build.Command, core.CompactError(msg.build.Err))
		case msg.build.Command != "":
			m.status = fmt.Sprintf("Clone e build completati (%s).", msg.build.Command)
		default:
			m.status = "Clone completato e aggiunto ai fetch recenti. Nessuna build riconosciuta."
		}
		m.screen = historyScreen
	} else {
		m.screen = resultScreen
	}
	return m, nil
}

// handleGlobalKey covers shortcuts that behave the same everywhere. Typing
// shortcuts like "q" are deliberately excluded from text-entry screens so a URL
// or a path containing the letter q can never quit the app.
func (m *model) handleGlobalKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if msg.Type == tea.KeyCtrlC {
		if m.cancelClone != nil {
			m.cancelClone()
		}
		m.quitting = true
		return tea.Quit, true
	}
	if (m.screen == urlScreen || m.screen == branchScreen || m.screen == destinationScreen) && msg.Type == tea.KeyCtrlV {
		return readClipboard(), true
	}
	if key.Matches(msg, appKeys.open) {
		if m.screen == resultScreen && m.result.target != "" {
			return openRepository(m.result.target), true
		}
		if m.screen == historyScreen && !m.searchActive && len(m.filtered) > 0 {
			entry := m.history[m.filtered[m.selected]]
			m.status = "Apro " + entry.Label() + "…"
			return openRepository(entry.Target), true
		}
		// Nothing to open here: don't claim the key.
		return nil, false
	}
	if key.Matches(msg, appKeys.copy) {
		if m.screen == resultScreen && m.result.target != "" {
			return copyPath(m.result.target), true
		}
		if m.screen == historyScreen && !m.searchActive && len(m.filtered) > 0 {
			return copyPath(m.history[m.filtered[m.selected]].Target), true
		}
		// Nothing to copy here: don't claim the key.
		return nil, false
	}
	if key.Matches(msg, appKeys.quit) {
		typing := (m.screen == historyScreen && m.searchActive) ||
			m.screen == urlScreen || m.screen == branchScreen || m.screen == destinationScreen
		if !typing {
			m.quitting = true
			return tea.Quit, true
		}
		// The user is typing in a text field: "q" is just a character.
		return nil, false
	}
	// Search and remove are only real shortcuts on the history screen while
	// not typing; elsewhere "/" or "ctrl+d" must reach the active input.
	if m.screen == historyScreen && !m.searchActive {
		if key.Matches(msg, appKeys.remove) {
			m.removeSelected()
			return nil, true
		}
		if key.Matches(msg, appKeys.search) {
			m.activateSearch()
			return textinput.Blink, true
		}
	}
	return nil, false
}

func (m *model) updateHistory(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.searchActive {
		switch keyMsg.Type {
		case tea.KeyEsc:
			m.searchActive = false
			m.searchInput.Blur()
			return m, nil
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				entry := m.history[m.filtered[m.selected]]
				m.searchActive = false
				m.searchInput.Blur()
				m.status = "Apro " + entry.Label() + "…"
				return m, openRepository(entry.Target)
			}
			return m, nil
		case tea.KeyUp:
			m.moveSelection(-1)
			return m, nil
		case tea.KeyDown:
			m.moveSelection(1)
			return m, nil
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.selected = 0
		m.refreshFilter()
		return m, cmd
	}

	switch keyMsg.Type {
	case tea.KeyUp:
		m.moveSelection(-1)
	case tea.KeyDown:
		m.moveSelection(1)
	case tea.KeyCtrlL:
		m.activateSearch()
		return m, textinput.Blink
	case tea.KeyCtrlD:
		m.removeSelected()
	case tea.KeyEnter:
		if len(m.filtered) > 0 {
			entry := m.history[m.filtered[m.selected]]
			m.status = "Apro " + entry.Label() + "…"
			return m, openRepository(entry.Target)
		}
	case tea.KeyRunes:
		switch string(keyMsg.Runes) {
		case "k":
			m.moveSelection(-1)
		case "j":
			m.moveSelection(1)
		case "n", "c":
			m.status = ""
			m.err = nil
			m.screen = urlScreen
			m.urlInput.SetValue("")
			m.urlInput.Focus()
			return m, textinput.Blink
		case "r":
			if m.store != nil {
				m.history = m.store.Load()
				m.refreshFilter()
			}
			m.status = "Cronologia aggiornata."
		case "u":
			// Update the selected clone in place with a fast-forward pull.
			if len(m.filtered) > 0 && m.pendingPull == "" {
				entry := m.history[m.filtered[m.selected]]
				m.pendingPull = entry.Target
				m.status = "Aggiorno " + entry.Label() + "…"
				return m, pullRepository(entry.Target)
			}
		case "/":
			m.activateSearch()
			return m, textinput.Blink
		}
	}
	return m, nil
}

// activateSearch focuses the history search field. It lives here so both the
// global shortcut path and the history screen path share one implementation.
func (m *model) activateSearch() {
	m.searchActive = true
	m.status = ""
	m.err = nil
	m.searchInput.Focus()
}

func (m *model) moveSelection(delta int) {
	if len(m.filtered) == 0 {
		return
	}
	m.selected = min(max(0, m.selected+delta), len(m.filtered)-1)
}

func (m *model) removeSelected() {
	if len(m.filtered) == 0 {
		return
	}
	entry := m.history[m.filtered[m.selected]]
	m.history = core.Remove(m.history, entry.Target)
	m.refreshFilter()
	if m.store == nil {
		m.status = "Voce rimossa."
		return
	}
	if err := m.store.Save(m.history); err != nil {
		m.status = "Voce rimossa, ma non riesco a salvare la cronologia."
	} else {
		m.status = "Voce rimossa: " + entry.Label()
	}
}

func (m *model) updateURL(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, appKeys.back) {
			m.screen = historyScreen
			m.urlInput.Blur()
			m.status = ""
			m.err = nil
			return m, nil
		}
		if key.Matches(msg, appKeys.enter) {
			if m.urlSubmitted && msg.String() == "enter" {
				// Enter is being held down (auto-repeat): ignore the extra
				// presses so one submission can never spawn parallel lookups.
				return m, nil
			}
			repo, err := core.ParseRepository(m.urlInput.Value())
			if err != nil {
				m.err = err
				return m, nil
			}
			m.urlSubmitted = true
			m.repository = repo
			m.err = nil
			m.status = ""
			m.screen = branchScreen
			m.urlInput.Blur()
			m.branchInput.SetValue("")
			m.branchInput.Focus()
			m.remoteBranches = nil
			m.branchCycling = -1
			// Warm the branch completion cache while the user reads the
			// screen; failures are silent, Tab just has nothing to suggest.
			return m, tea.Batch(textinput.Blink, fetchRemoteBranches(repo.URL))
		}
	default:
		// Backspace/delete/reset re-arms Enter after a held-key repeat.
		switch msg.(type) {
		case tea.KeyMsg:
			km := msg.(tea.KeyMsg)
			if km.Type != tea.KeyRunes {
				m.urlSubmitted = false
			}
		default:
			m.urlSubmitted = false
		}
	}
	var cmd tea.Cmd
	m.urlInput, cmd = m.urlInput.Update(msg)
	return m, cmd
}

// updateBranch lets the user optionally type a branch to clone. Leaving the
// field empty clones the remote's default branch, so Enter always continues.
func (m *model) updateBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, appKeys.back) {
			m.screen = urlScreen
			m.branchInput.Blur()
			m.urlInput.Focus()
			m.err = nil
			return m, textinput.Blink
		}
		if key.Matches(msg, appKeys.enter) {
			branch := strings.TrimSpace(m.branchInput.Value())
			if strings.ContainsAny(branch, " \t\n") {
				m.err = fmt.Errorf("il branch non può contenere spazi")
				return m, nil
			}
			m.err = nil
			m.status = ""
			m.screen = destinationScreen
			m.branchInput.Blur()
			m.dirInput.Focus()
			m.suggestionIndex = -1
			return m, textinput.Blink
		}
		if msg.Type == tea.KeyTab {
			// Tab completes from the remote's branch list; pressing it again
			// cycles through every match for the current prefix.
			matches := core.BranchCompletions(m.remoteBranches, m.branchInput.Value())
			if len(matches) > 0 {
				m.branchCycling = (m.branchCycling + 1) % len(matches)
				m.branchInput.SetValue(matches[m.branchCycling])
				m.branchInput.CursorEnd()
			} else if len(m.remoteBranches) == 0 {
				m.status = "Elenco branch non disponibile (repo privato o rete lenta)."
			}
			return m, nil
		}
	}
	m.branchCycling = -1
	var cmd tea.Cmd
	m.branchInput, cmd = m.branchInput.Update(msg)
	return m, cmd
}

func (m *model) updateDestination(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmOverwrite {
			switch {
			case key.Matches(msg, appKeys.enter):
				m.confirmOverwrite = false
				return m.startClone(true)
			case key.Matches(msg, appKeys.back):
				m.confirmOverwrite = false
				m.status = "Sovrascrittura annullata."
			}
			return m, nil
		}
		if key.Matches(msg, appKeys.back) {
			// Back from the destination returns to the branch step, not all the
			// way to the URL: the repository is still the one being cloned.
			m.screen = branchScreen
			m.dirInput.Blur()
			m.branchInput.Focus()
			m.err = nil
			return m, textinput.Blink
		}
		if key.Matches(msg, appKeys.enter) {
			target, err := core.DestinationPath(m.dirInput.Value(), m.repository)
			if err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			if core.NonEmptyDir(target) {
				m.confirmOverwrite = true
				return m, nil
			}
			return m.startClone(false)
		}
		if msg.Type == tea.KeyTab {
			if suggestion := m.nextSuggestion(); suggestion != "" {
				m.dirInput.SetValue(suggestion)
				m.dirInput.CursorEnd()
				m.status = "Suggerimento: " + suggestion
			}
			return m, nil
		}
		if msg.String() == "s" {
			// Toggle a shallow clone: only the latest commit is downloaded,
			// which is dramatically faster on big repositories.
			if m.depth > 0 {
				m.depth = 0
				m.status = "Clone completo: verrà scaricata tutta la storia."
			} else {
				m.depth = 1
				m.status = "Clone shallow attivo: solo l'ultimo commit."
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.dirInput, cmd = m.dirInput.Update(msg)
	return m, cmd
}

func (m *model) startClone(overwrite bool) (tea.Model, tea.Cmd) {
	m.status = ""
	m.screen = cloningScreen
	m.startedAt = time.Now()
	m.dirInput.Blur()
	m.cloneBranch = strings.TrimSpace(m.branchInput.Value())
	m.progressLog = nil
	m.progressChan = make(chan string, 256)
	cmd, cancel := startFetch(m.repository, m.dirInput.Value(), m.cloneBranch, overwrite, m.noBuild, m.depth, m.store, m.progressChan)
	m.cancelClone = cancel
	return m, tea.Batch(m.spinner.Tick, progressTick(), waitForProgress(m.progressChan), cmd)
}

// nextSuggestion cycles through the most recent destination directories. It
// uses a pointer receiver so the cursor advances across calls.
func (m *model) nextSuggestion() string {
	suggestions := core.RecentDirectories(m.history, 8)
	if len(suggestions) == 0 {
		return ""
	}
	m.suggestionIndex = (m.suggestionIndex + 1) % len(suggestions)
	return suggestions[m.suggestionIndex]
}

func (m *model) updateResult(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, appKeys.enter):
			if m.result.err == nil && m.result.target != "" {
				return m, openRepository(m.result.target)
			}
			m.screen = historyScreen
			m.err = nil
			return m, textinput.Blink
		case key.Matches(keyMsg, appKeys.back):
			m.screen = destinationScreen
			m.confirmOverwrite = false
			m.dirInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func copyPath(target string) tea.Cmd {
	return func() tea.Msg {
		return copiedMsg{err: clipboard.WriteAll(target)}
	}
}
