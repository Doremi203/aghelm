package ui

import (
	"testing"

	uisurface "github.com/Doremi203/aghelm/internal/ui/surface"
	tea "github.com/charmbracelet/bubbletea"
)

// A click off a dismissible card closes it the way esc does, through the
// same router: the surface says it is dismissible, mouse.go does not guess.
func TestClickOffADismissibleSurfaceClosesIt(t *testing.T) {
	m := buildModel(t)
	m.openHelp()
	if !m.helpOpen() {
		t.Fatal("test setup: help should open")
	}
	m.prepareFrame()
	press := tea.MouseMsg{X: 0, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	release := tea.MouseMsg{X: 0, Y: 0, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft}
	m.Update(press)
	m.Update(release)
	if m.helpOpen() || !m.onList() {
		t.Fatalf("help open=%t screen = %v", m.helpOpen(), m.screenID())
	}
}

// leaveSpy records whether pop ran its Leave, once, with the host.
type leaveSpy struct {
	uisurface.Surface
	left int
}

func (s *leaveSpy) Leave(h uisurface.Host) {
	if h == nil {
		panic("Leave ran with no host")
	}
	s.left++
}

// openedHelp is a real surface to stack: help, opened and taken off the top.
func openedHelp(t *testing.T, m *Model) uisurface.Surface {
	t.Helper()
	m.openHelp()
	top := m.surfaces[len(m.surfaces)-1]
	m.surfaces = m.surfaces[:len(m.surfaces)-1]
	return top
}

// pop runs a surface's Leave exactly once, and clearing a stack of
// surfaces runs each one, top first.
func TestPopRunsLeaveOnce(t *testing.T) {
	m := buildModel(t)
	spy := &leaveSpy{Surface: openedHelp(t, m)}
	m.push(spy)
	m.Pop()
	m.Pop()
	if spy.left != 1 {
		t.Fatalf("Leave ran %d times, want 1", spy.left)
	}
	m.push(spy)
	m.push(openedHelp(t, m))
	m.clearSurfaces()
	if spy.left != 2 || len(m.surfaces) != 0 {
		t.Fatalf("after clearSurfaces: Leave ran %d times, %d surfaces left", spy.left, len(m.surfaces))
	}
}

// pop clears the slot it leaves behind, so the backing array does not keep
// a closed surface and its state alive until the slot is reused.
func TestPopReleasesTheClosedSurface(t *testing.T) {
	m := buildModel(t)
	m.push(openedHelp(t, m))
	m.push(openedHelp(t, m))
	m.Pop()
	m.clearSurfaces()
	for i, s := range m.surfaces[:cap(m.surfaces)] {
		if s != nil {
			t.Fatalf("slot %d still holds %T after it was popped", i, s)
		}
	}
}
