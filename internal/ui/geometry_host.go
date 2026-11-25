package ui

import (
	uigeometry "github.com/Doremi203/aghelm/internal/ui/geometry"
	uigeometrypanes "github.com/Doremi203/aghelm/internal/ui/geometry/panes"
	"github.com/charmbracelet/lipgloss"
)

// geometryHost is what the geometry package asks the root: the terminal box
// and what its chrome costs.
type geometryHost struct{ m *Model }

func (h geometryHost) Layout() uigeometry.Layout {
	return uigeometry.Layout{
		Width:           h.m.layout.width,
		Height:          h.m.layout.height,
		HeaderRows:      h.m.headerRows(),
		FooterRows:      func() int { return lipgloss.Height(h.m.viewFooter()) },
		QuickFooterRows: h.quickFooterRows,
		QuickBar:        h.m.quick() != nil,
	}
}

// quickFooterRows is the taller footer the quick bar paints, or zero when no
// bar is up and the resting footer is the painted one.
func (h geometryHost) quickFooterRows() int {
	if h.m.quick() == nil {
		return 0
	}
	return lipgloss.Height(h.m.listFooter())
}

// paneSizing is the root as the geometry runtime reaches it: what a shrink
// costs a tool, and the effect lane a pin rides.
type paneSizing struct{ *Model }

func (r paneSizing) FitsHeight(tool string) bool { return r.services.cfg.Tools[tool].FitsHeight }

func (r paneSizing) Queue(request uigeometrypanes.Resize) { r.queueGeometry(request) }

var _ uigeometrypanes.Root = paneSizing{}
