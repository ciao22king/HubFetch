package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/ciao22king/GoFetch/internal/core"
)

// View renders the current screen. Frames are built into a strings.Builder
// with a generous initial capacity to avoid repeated reallocation, and every
// lipgloss style is a package-level constant so no style is rebuilt per frame.
func (m model) View() string {
	if m.quitting {
		return "A presto.\n"
	}

	var body strings.Builder
	body.Grow(4096)

	switch m.screen {
	case historyScreen:
		body.WriteString(m.viewHistory())
	case urlScreen:
		body.WriteString(m.viewURL())
	case branchScreen:
		body.WriteString(m.viewBranch())
	case destinationScreen:
		body.WriteString(m.viewDestination())
	case cloningScreen:
		body.WriteString(m.viewCloning())
	case resultScreen:
		body.WriteString(m.viewResult())
	}

	var content strings.Builder
	content.WriteString(logoStyle.Render("  GoFetch "))
	content.WriteString(subtitleStyle.Render("  v" + m.version))
	content.WriteByte('\n')
	content.WriteByte('\n')
	content.WriteString(body.String())
	if m.err != nil {
		content.WriteByte('\n')
		content.WriteString(errorStyle.Render("  " + m.err.Error()))
	}
	if m.status != "" {
		content.WriteByte('\n')
		content.WriteString(labelStyle.Render("  " + m.status))
	}
	content.WriteByte('\n')
	content.WriteString(m.helpView())

	return frameStyle.Render(content.String())
}

// helpBubble is a single package-level help model: creating one per frame was
// an unnecessary allocation, and its styles are set once here.
var helpBubble = func() help.Model {
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(muted)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(muted)
	return h
}()

func (m model) helpView() string {
	helpBubble.Width = max(m.width-8, 30)
	return helpBubble.View(m)
}

func (m model) ShortHelp() []key.Binding {
	switch m.screen {
	case historyScreen:
		if m.searchActive {
			return []key.Binding{appKeys.enter, appKeys.back}
		}
		return []key.Binding{
			appKeys.quit,
			key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "nuovo fetch")),
			appKeys.open,
			appKeys.copy,
			appKeys.remove,
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "aggiorna")),
			appKeys.search,
		}
	case urlScreen:
		return []key.Binding{appKeys.enter, appKeys.back, key.NewBinding(key.WithKeys("ctrl+v"), key.WithHelp("ctrl+v", "incolla"))}
	case branchScreen:
		return []key.Binding{
			appKeys.enter,
			appKeys.back,
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "completa")),
			key.NewBinding(key.WithKeys("ctrl+v"), key.WithHelp("ctrl+v", "incolla")),
		}
	case destinationScreen:
		if m.confirmOverwrite {
			return []key.Binding{appKeys.enter, appKeys.back}
		}
		return []key.Binding{
			appKeys.enter,
			appKeys.back,
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "shallow")),
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cartelle recenti")),
			key.NewBinding(key.WithKeys("ctrl+v"), key.WithHelp("ctrl+v", "incolla")),
		}
	case cloningScreen:
		return []key.Binding{appKeys.back}
	case resultScreen:
		return []key.Binding{
			appKeys.enter,
			appKeys.back,
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "cronologia")),
		}
	default:
		return []key.Binding{appKeys.quit}
	}
}

func (m model) FullHelp() [][]key.Binding { return [][]key.Binding{m.ShortHelp()} }

func (m model) viewHistory() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Fetch recenti") + "\n\n")

	if len(m.history) == 0 {
		b.WriteString(subtitleStyle.Render("  Nessun repository recuperato."))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("  Premi n o c per incollare un URL GitHub o GitLab."))
		return b.String()
	}

	if m.searchActive {
		b.WriteString("  " + m.searchInput.View() + "\n\n")
	}

	visible := min(12, len(m.filtered))
	// The window of rows always centers on the selection, so long histories
	// scroll without rebuilding the whole list every keystroke.
	start := max(0, m.selected-visible/2)
	end := min(len(m.filtered), start+visible)
	if end-start < visible {
		start = max(0, end-visible)
	}

	for position := start; position < end; position++ {
		index := m.filtered[position]
		entry := m.history[index]
		name := entry.Label()
		meta := fmt.Sprintf("%s · %s · %s", entry.Provider, entry.Target, core.RelativeTime(entry.FetchedAt))
		pending := m.pendingPull == entry.Target
		b.WriteString(historyRow(name, meta, pending, position == m.selected) + "\n")
	}

	shown := len(m.filtered)
	if shown > visible {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  ... altri %d risultati", shown-visible)))
		b.WriteByte('\n')
	}
	if m.searchActive && shown == 0 {
		b.WriteString(subtitleStyle.Render("  Nessun risultato per questa ricerca."))
		b.WriteByte('\n')
	}
	return b.String()
}

// historyRow renders one history line. Only the selected row gets a full
// lipgloss pass; the others use cheap ANSI constants, which keeps the cost of
// redrawing a 100-entry list negligible.
func historyRow(name, meta string, pending, selected bool) string {
	symbol := "  "
	if pending {
		symbol = "↻ "
	}
	if !selected {
		return "  " + symbol + name + "\n      " + dimANSI + meta + resetANSI
	}
	label := pillStyle.Render(symbol+name) + " " + rowMetaStyle.Render(meta)
	return selectedRowStyle.Render(strings.TrimLeft(label, " "))
}

const (
	dimANSI   = "\x1b[38;5;244m"
	resetANSI = "\x1b[0m"
)

var selectedRowStyle = lipgloss.NewStyle().
	Foreground(ink).
	Background(lipgloss.Color("#1E293B")).
	Padding(0, 1)

func providerPill(provider string) string {
	return badgeStyle.Render(" " + provider + " ")
}

func (m model) viewURL() string {
	return cardStyle.Render(
		titleStyle.Render("Nuovo fetch") + "\n\n" +
			labelStyle.Render("URL del repository") + "\n" +
			m.urlInput.View() + "\n\n" +
			subtitleStyle.Render("Incolla un link GitHub o GitLab, oppure premi ctrl+v."),
	)
}

func (m model) viewBranch() string {
	b := strings.Builder{}
	b.WriteString(cardStyle.Render(
		titleStyle.Render("Branch") + "\n\n" +
			labelStyle.Render("Repository") + "\n" +
			targetStyle.Render(m.repository.Provider+" / "+m.repository.Name) + "\n\n" +
			labelStyle.Render("Branch da clonare (premi Enter per quello predefinito)") + "\n" +
			m.branchInput.View(),
	))
	switch {
	case len(m.remoteBranches) > 0:
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  %d branch disponibili — Tab completa.", len(m.remoteBranches))))
	case m.remoteBranches == nil:
		b.WriteString("\n")
		b.WriteString(progressStyle.Render("  Sto leggendo i branch remoti…"))
	}
	return b.String()
}

func (m model) viewDestination() string {
	parent := m.dirInput.Value()
	repo := m.repository
	target, err := core.DestinationPath(parent, repo)
	if err != nil {
		target = "(percorso non valido)"
	}

	var b strings.Builder
	b.WriteString(cardStyle.Render(
		titleStyle.Render("Destinazione") + "\n\n" +
			labelStyle.Render("Cartella padre") + "\n" +
			m.dirInput.View() + "\n\n" +
			labelStyle.Render("Verrà creato in") + "\n" +
			targetStyle.Render(target) + "\n\n" +
			subtitleStyle.Render("Tab suggerisce le cartelle usate di recente."),
	))

	// Status pills: shallow mode and build opt-out are easy to forget, so they
	// stay visible right under the card.
	var pills []string
	if m.depth > 0 {
		pills = append(pills, shallowPill.Render(fmt.Sprintf(" shallow: solo ultimi %d commit ", m.depth)))
	} else {
		pills = append(pills, pillStyle.Render(" clone completo "))
	}
	if m.noBuild {
		pills = append(pills, pillStyle.Render(" build disattivata "))
	}
	if len(pills) > 0 {
		b.WriteString("\n" + strings.Join(pills, " "))
	}

	if m.confirmOverwrite {
		b.WriteString("\n\n")
		b.WriteString(dangerStyle.Render("⚠ La cartella ") + targetStyle.Render(target) + dangerStyle.Render(" esiste già e non è vuota.") + "\n")
		b.WriteString(labelStyle.Render("  Invio per sovrascrivere · Esc per annullare"))
	}
	return b.String()
}

func (m model) viewCloning() string {
	elapsed := time.Since(m.startedAt).Round(time.Second)
	lines := []string{
		m.spinner.View() + " Cloning " + providerPill(m.repository.Provider) + " " + titleStyle.Render(m.repository.Name),
		labelStyle.Render("  Branch: ") + targetStyle.Render(orDefault(m.cloneBranch, "default")) +
			labelStyle.Render("   Tempo: ") + targetStyle.Render(elapsed.String()),
		"",
	}
	for _, line := range tailLines(m.progressLog, progressWindow(m.height)) {
		lines = append(lines, progressStyle.Render(truncateLine(line, max(m.width-6, 40))))
	}
	if len(m.progressLog) == 0 {
		lines = append(lines, subtitleStyle.Render("  In attesa della risposta del remote…"))
	}
	lines = append(lines, "", helpStyle.Render("  Esc annulla il clone"))
	return strings.Join(lines, "\n")
}

// progressWindow sizes the visible progress log to the terminal height so the
// footer never scrolls off-screen; small terminals see fewer lines.
func progressWindow(height int) int {
	if height <= 0 {
		return 8
	}
	return max(3, min(15, height-16))
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (m model) viewResult() string {
	result := m.result
	var b strings.Builder
	if result.err == nil {
		b.WriteString(successStyle.Render("✔ Clone completato") + "\n\n")
		b.WriteString(labelStyle.Render("Destinazione") + "\n")
		b.WriteString(targetStyle.Render(result.target) + "\n")
		if result.build.Command != "" {
			b.WriteString("\n")
			b.WriteString(labelStyle.Render("Build automatica") + "\n")
			if result.build.Err != nil {
				b.WriteString(errorStyle.Render(result.build.Command + ": " + core.CompactError(result.build.Err)))
			} else {
				b.WriteString(successStyle.Render(result.build.Command + " completata in " + result.build.Duration.Round(time.Second).String()))
			}
		} else if !result.build.Skipped {
			b.WriteString("\n" + subtitleStyle.Render("Nessuna build automatica"))
		}
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("  Invio o ctrl+o apre la cartella · q o esc torna alla cronologia"))
		return b.String()
	}

	message := "Il fetch non è riuscito."
	if result.err != nil {
		message = result.err.Error()
	}
	b.WriteString(dangerStyle.Render("✖ Operazione interrotta") + "\n\n")
	b.WriteString(errorStyle.Render(message) + "\n\n")
	if result.target != "" {
		b.WriteString(labelStyle.Render("Destinazione") + "\n" + targetStyle.Render(result.target) + "\n\n")
	}
	if output := compactTail(result.output, 10); output != "" {
		b.WriteString(labelStyle.Render("Output git") + "\n")
		b.WriteString(progressStyle.Render(output) + "\n\n")
	}
	b.WriteString(helpStyle.Render("  Esc per modificare l'URL · ctrl+y copia il percorso · ctrl+o apre la cartella"))
	return b.String()
}

func compactTail(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return strings.Join(lines, "\n")
}

// tailLines returns the last n entries of lines without copying more than the
// window itself.
func tailLines(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

// truncateLine trims a progress line to width runes for single-column display.
func truncateLine(line string, width int) string {
	line = strings.TrimSpace(line)
	if utf8Len(line) <= width {
		return line
	}
	runes := []rune(line)
	return string(runes[:width-1]) + "…"
}

func utf8Len(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}
