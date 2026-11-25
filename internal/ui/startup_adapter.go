package ui

import (
	"github.com/Doremi203/aghelm/internal/status"
	uiframe "github.com/Doremi203/aghelm/internal/ui/frame"
	uistartup "github.com/Doremi203/aghelm/internal/ui/startup"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// startupState is the loader, the wordmark sweep and the boot ring: the
// package owns the timing and the frame counter, the root the fields the
// rail and layout read every frame.
type startupState = uistartup.State

func (m *Model) routeStartupMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	cmd, ok := m.startup.Update(startupRoot{m}, msg)
	if !ok {
		return nil, nil, false
	}
	return routed(m, cmd)
}

// startupRoot is the root as the startup animation reads it.
type startupRoot struct{ *Model }

func (h startupRoot) LoaderNeeded() bool { return h.needsLoaderTick() }

func (m *Model) startStartupTick() tea.Cmd { return m.startup.StartTick(startupRoot{m}) }

func (m *Model) bannerTick() tea.Cmd { return m.startup.BannerTick() }

// showBanner reports whether the terminal leaves the tracked-out wordmark
// room beside the fleet readings; a narrow one gets the name set plainly.
func (m *Model) showBanner() bool {
	return uistartup.Fits(m.layout.width, uiframe.RailGutter)
}

// headerRows is how many rows the header occupies, which the body height
// and every mouse hit-test are measured against.
func (m *Model) headerRows() int {
	if m.prefs.hideHeader {
		return 0
	}
	return uistartup.Rows
}

// viewBanner draws the wordmark, lit by a highlight that sweeps left to
// right during the intro and then rests just past the end of the word.
func (m *Model) viewBanner() []string {
	return uistartup.Line(m.startup.BannerPhase, uiframe.RailGutter, m.showBanner(), uitheme.Current())
}

func (m *Model) reviewNeedsLoader() bool {
	state := m.review.Snapshot()
	if m.base != baseReview || !state.Active {
		return false
	}
	if state.Loading && len(state.Set.Files) == 0 {
		return true
	}
	fd := m.currentFileDiff()
	return fd != nil && !fd.Loaded() && !m.diffFileHidden(fd)
}

func (m *Model) needsLoaderTick() bool {
	return m.startup.Booting || m.rail.HasStatus(status.Starting) || m.reviewNeedsLoader() || m.rail.HasWorkingLoader()
}
