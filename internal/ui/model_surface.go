package ui

import (
	uisurface "github.com/Doremi203/aghelm/internal/ui/surface"
	tea "github.com/charmbracelet/bubbletea"
)

// screen is what is on the terminal: the base screen and the surfaces over
// it, read from the top.
type screen struct {
	base     baseScreen
	surfaces []uisurface.Surface
}

// baseScreen is the screen under the stack: the list, or Review in its
// place.
type baseScreen uint8

const (
	baseList baseScreen = iota
	baseReview
)

// screenID names where the user is, for a completion that must land on the
// screen it was asked from: the base and how many surfaces are up, so a
// card opened since the request keeps it.
type screenID struct {
	base  baseScreen
	depth int
}

func (m *Model) screenID() screenID {
	return screenID{base: m.base, depth: len(m.surfaces)}
}

// mouseTaker is the optional capability of a surface that is not a card:
// it takes every mouse event itself, and the root draws no card edge or
// backdrop around it.
type mouseTaker interface {
	Mouse(h uisurface.Host, msg tea.MouseMsg) tea.Cmd
}

func (m *Model) push(s uisurface.Surface) { m.surfaces = append(m.surfaces, s) }

func (m *Model) Pop() {
	n := len(m.surfaces)
	if n == 0 {
		return
	}
	top := m.surfaces[n-1]
	m.surfaces[n-1] = nil
	m.surfaces = m.surfaces[:n-1]
	top.Leave(m)
}

// clearSurfaces pops every surface, top first; leaving the screen they sit
// on does this.
func (m *Model) clearSurfaces() {
	for len(m.surfaces) > 0 {
		m.Pop()
	}
}

func (m *Model) top() (uisurface.Surface, bool) {
	if n := len(m.surfaces); n > 0 {
		return m.surfaces[n-1], true
	}
	return nil, false
}

// surfaceOf finds the mounted surface of one type, for a completion that
// must land on the surface that asked and nowhere else.
func surfaceOf[S uisurface.Surface](m *Model) (S, bool) {
	for i := len(m.surfaces) - 1; i >= 0; i-- {
		if s, ok := m.surfaces[i].(S); ok {
			return s, true
		}
	}
	var none S
	return none, false
}

func (m *Model) surfaceKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	for i := len(m.surfaces) - 1; i >= 0; i-- {
		if handled, cmd := m.surfaces[i].Key(m, msg); handled {
			return cmd, true
		}
	}
	return nil, false
}

func (m *Model) surfaceView(frame string) string {
	for _, s := range m.surfaces {
		frame = s.View(m, frame)
	}
	return frame
}
