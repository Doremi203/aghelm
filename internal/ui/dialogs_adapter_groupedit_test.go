package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Editing a group opens a card, and a click steps what it lands on before
// the edit saves.
func TestGroupEditIsACard(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 40
	defineEnvs(t, m, "work")
	if err := m.services.store.CreateGroup("team", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	m.applyCmd(t, m.refreshCmd())
	m.selectGroupRow(t, "team")
	m.openRename()
	if !m.groupFormOpen() || m.groupSurface().Edit() != "team" {
		t.Fatalf("r on a group did not open its edit card")
	}
	for _, label := range []string{"path ", "worktree ", "base ", "env "} {
		paintedCell(t, m, label)
	}
	x, y := paintedCell(t, m, "env ")
	m = leftClick(m, x, y)
	m = leftClick(m, x, y)
	x, y = paintedCell(t, m, "worktree ")
	m = leftClick(m, x, y)
	m = leftClick(m, x, y)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.applyCmd(t, cmd)
	groups, err := m.services.store.Groups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Environment != "work" || groups[0].Worktree != "on" {
		t.Fatalf("saved group = %+v, want env work and worktree on", groups)
	}
	if m.groupFormOpen() {
		t.Fatalf("the edit card stayed open after a save")
	}
}
