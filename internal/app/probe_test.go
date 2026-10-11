package app

import (
"testing"

tea "github.com/charmbracelet/bubbletea"

"github.com/ciao22king/GoFetch/internal/core"
)

func TestProbeUpdate(t *testing.T) {
m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
m2 := upd.(model)
t.Logf("PROBE1 Update q on url screen: quitting=%v url=%q", m2.quitting, m2.urlInput.Value())

mh := newTestModel(t, Options{})
mh.history = []core.Entry{{Repository: core.Repository{Name: "alpha"}, Target: "/code/alpha"}}
mh.refreshFilter()
updH, cmdH := mh.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
m3 := updH.(model)
t.Logf("PROBE2 Update ctrl+l: searchActive=%v cmdNil=%v", m3.searchActive, cmdH == nil)
}
