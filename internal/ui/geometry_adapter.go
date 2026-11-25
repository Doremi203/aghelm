package ui

import (
	uieffects "github.com/Doremi203/aghelm/internal/ui/effects"
	uigeometry "github.com/Doremi203/aghelm/internal/ui/geometry"
	uigeometrypanes "github.com/Doremi203/aghelm/internal/ui/geometry/panes"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) geometryLayout() uigeometry.Layout { return geometryHost{m}.Layout() }

func (m *Model) paneTargetSize() (int, int) { return m.geometryLayout().PaneSize() }

func (m *Model) listChromeRows() int { return m.geometryLayout().ChromeRows() }

func (m *Model) listBodyHeight() int { return m.geometryLayout().BodyHeight() }

func (m *Model) routeResizeMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	size, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return nil, nil, false
	}
	// Resuming from a tmux attach re-sends the current size unchanged; only a
	// real resize needs the per-session tmux resize calls.
	if size.Width == m.layout.width && size.Height == m.layout.height {
		return routed(m, nil)
	}
	m.layout.width = size.Width
	m.layout.height = size.Height
	// Re-assert the terminal backdrop: a reattach or a fresh outer terminal
	// delivers a size message and may carry stale colors.
	SyncTerminalColors()
	m.publishPaneSize()
	m.resizeSessions()
	m.syncFormFieldWidths()
	return routed(m, nil)
}

func (m *Model) applyGeometryEffect(result uigeometrypanes.Result, err error) {
	m.layout.panes.Apply(result)
	if err != nil {
		m.reportErr(err.Error())
	}
}

// queueGeometry queues a resize, dropping what the lane already carries and
// folding into the geometry job queued just before it, never across an
// accepted mutation.
func (m *Model) queueGeometry(request uigeometrypanes.Resize) {
	request = uigeometrypanes.Effective(request, m.queuedGeometry())
	if request.Empty() {
		return
	}
	if n := len(m.effects.Main.Pending); n > 0 {
		if previous, ok := m.effects.Main.Pending[n-1].Request.(uigeometrypanes.ResizeRequest); ok {
			request = uigeometrypanes.Merge(previous.Resize, request)
			m.effects.Main.Pending = m.effects.Main.Pending[:n-1]
		}
	}
	m.enqueueEffect(uigeometrypanes.ResizeRequest{Resize: request}, 0, false)
}

// queuedGeometry reads the resizes already waiting on the lane, in the order
// the lane will run them.
func (m *Model) queuedGeometry() []uigeometrypanes.Resize {
	jobs := append([]*uieffects.Job(nil), m.effects.Main.Pending...)
	if m.effects.Main.Active != nil {
		jobs = append([]*uieffects.Job{m.effects.Main.Active}, jobs...)
	}
	queued := make([]uigeometrypanes.Resize, 0, len(jobs))
	for _, job := range jobs {
		if request, ok := job.Request.(uigeometrypanes.ResizeRequest); ok {
			queued = append(queued, request.Resize)
		}
	}
	return queued
}

// resizeSessions syncs every live session's tmux window to the body's box, so
// an attach opens at the size the manager last painted.
func (m *Model) resizeSessions() {
	width, height := m.paneTargetSize()
	sessions := make([]uigeometrypanes.Session, 0, len(m.workspace.sessions))
	for _, sess := range m.workspace.sessions {
		sessions = append(sessions, uigeometrypanes.Session{
			ID:        sess.ID,
			Tool:      sess.Tool,
			AltScreen: m.workspace.panes[sess.ID].AltScreen,
			Archived:  sess.Archived,
		})
	}
	m.layout.panes.QueueAll(paneSizing{m}, uigeometrypanes.Box{Width: width, Height: height}, sessions, m.workspace.panes)
}

// markFreshPane queues one exact size pin for a session whose window this run
// just created.
func (m *Model) markFreshPane(id string) { m.layout.panes.MarkFresh(id) }

// publishPaneSize records the box for the launch paths that run without a
// manager.
func (m *Model) publishPaneSize() {
	if m.layout.width <= 0 {
		return
	}
	width, height := m.paneTargetSize()
	m.layout.panes.Publish(paneSizing{m}, width, height)
}

// adoptPaneGeom takes a session off the geometry cache, so its next pin is
// exact: an attach resized its window to the whole terminal and tmux keeps
// that size on detach.
func (m *Model) adoptPaneGeom(id string) {
	m.layout.panes.Forget(id)
	width, height := m.paneTargetSize()
	m.queueGeometry(uigeometrypanes.Resize{Targets: []uigeometrypanes.PaneResize{{ID: id, Size: [2]int{width, height}}}})
}
