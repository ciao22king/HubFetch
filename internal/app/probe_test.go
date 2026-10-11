package app

import (
"testing"

tea "github.com/charmbracelet/bubbletea"
)

func TestProbeUpdate(t *testing.T) {
m := newTestModel(t, Options{InitialURL: "https://github.com/owner/project"})
updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
m2 := updated.(model)
t.Logf("PROBE Update q: cmdNotNil=%v quitting=%v urlValue=%q", cmd != nil, m2.quitting, m2.urlInput.Value())
}
