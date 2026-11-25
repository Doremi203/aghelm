package ui

import (
	"github.com/Doremi203/aghelm/internal/keybind"
	uirail "github.com/Doremi203/aghelm/internal/ui/rail"
	tea "github.com/charmbracelet/bubbletea"
)

// routeQuit is the one place that decides ctrl+c. Update asks it before a
// drain swallows keys, and handleKey asks it again for a key a click
// presses, so no surface keeps its own idea of quit. ctrl+c quits from
// anywhere, card or no card; a drain already under way also leaves on the
// quit key.
func (m *Model) routeQuit(msg tea.KeyMsg) bool {
	name := keybind.Normalize(msg.String())
	if name == "ctrl+c" {
		return true
	}
	return m.effects.Quitting && m.services.listKeys.Binding(keybind.Quit).Has(name)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.routeQuit(msg) {
		return m.requestQuit()
	}
	if cmd, ok := m.surfaceKey(msg); ok {
		return m, cmd
	}
	if m.base == baseReview {
		m.gens.foreground++
		return m.handleDiffKey(msg)
	}
	return m.handleRailKey(msg)
}

func (m *Model) railKeyContext() uirail.KeyContext {
	return uirail.KeyContext{ListKeys: m.services.listKeys}
}
