package ui

import (
	"github.com/Doremi203/aghelm/internal/keybind"
	uieffects "github.com/Doremi203/aghelm/internal/ui/effects"
	uinotices "github.com/Doremi203/aghelm/internal/ui/notices"
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uiterminal "github.com/Doremi203/aghelm/internal/ui/terminal"
	tea "github.com/charmbracelet/bubbletea"
)

const docsURL = uinotices.DocsURL

var _ columnClicker = (*uinotices.Surface)(nil)

// noticesPanel is the messages modal: uinotices.Panel owns the list it
// builds, the cursor, the body scroll, the dismissed and seen sets. The root
// keeps a refused config import.
type noticesPanel struct {
	uinotices.Panel
	uinotices.Log
}

// noticeSources are the root facts the notice list is built from, copied so
// the panel never reaches into the store or the key tables itself.
func (m *Model) noticeSources() uinotices.Sources {
	return uinotices.Sources{
		Store:             m.services.store != nil,
		ConfigImportError: m.notices.ImportError,
	}
}

func (m *Model) activeNotices() []uinotices.Notice {
	return m.notices.Active(m.noticeSources())
}

// noticesFrame is the terminal and the status bar as the panel's tail
// shows them.
func (m *Model) noticesFrame() uinotices.Frame {
	return uinotices.Frame{
		CloseKey:  m.services.listKeys.Binding(keybind.Messages),
		GlyphMode: m.prefs.glyphMode,
		Width:     m.layout.width,
		Height:    m.layout.height,
		Status:    m.statusRow(),
		Theme:     uinotices.PanelTheme(),
	}
}

// openNotices shows the panel on the notice with id, or on the first. An
// open panel moves to it instead of stacking a second.
func (m *Model) openNotices(id string) {
	if s, ok := surfaceOf[*uinotices.Surface](m); ok {
		s.Panel().Open(m.activeNotices(), id)
		m.seeShownNotice(s)
		return
	}
	m.push(uinotices.NewSurface(noticesRoot{m}, m.notices.Panel, id))
}

// lastSeenVersionSetting records the version the setup screen was shown on,
// so it opens on a store's first launch only.
const lastSeenVersionSetting = "last_seen_version"

// openStartupGreeting opens the setup screen on the first launch against a
// store, and records this version so the next launch does not. A failed
// read greets nothing; a failed write reports and greets nothing, so the
// next launch tries again.
func (m *Model) openStartupGreeting() {
	seen, err := m.services.store.Setting(lastSeenVersionSetting)
	if err != nil || seen != "" {
		return
	}
	if err := m.services.store.SetSetting(lastSeenVersionSetting, m.version); err != nil {
		m.reportErr(err.Error())
		return
	}
	m.openSetup(true)
}

func (m *Model) noticesOpen() bool {
	_, open := surfaceOf[*uinotices.Surface](m)
	return open
}

// paintNoticesCard floats the panel on the backdrop and records its body,
// the rows between the edges and a column in from each, and the keys its
// bottom edge names.
func (m *Model) paintNoticesCard(card []string, foot []presentation.PaintedKey) string {
	frame := m.centerOnBackdrop(card)
	rect := &m.layout.card
	rect.bodyTop, rect.bodyLeft, rect.bodyWidth = rect.top+1, rect.left+2, rect.right-rect.left-4
	m.placePainted(foot, rect.left, rect.bottom-1)
	return frame
}

func (m *Model) takeForeground() { m.gens.foreground++ }

func (m *Model) dismissNotice(id string) {
	if id == "" || m.services.store == nil || m.noticeDismissQueued(id) || m.notices.Dismissed(id) {
		return
	}
	m.notices.Dismiss(id)
	m.enqueueEffect(uinotices.DismissRequest{
		ID:            id,
		ForegroundGen: m.gens.foreground,
		Modal:         m.noticesOpen(),
	}, 0, false)
}

// seeNotice marks the notice the open panel shows as read and persists it.
func (m *Model) seeNotice(id string) {
	if id == "" || m.services.store == nil || m.notices.Seen(id) {
		return
	}
	m.notices.See(id)
	m.enqueueEffect(uinotices.SeenRequest{ID: id}, 0, false)
}

// seeShownNotice marks read the notice a cursor the root moved now shows.
func (m *Model) seeShownNotice(s *uinotices.Surface) {
	m.seeNotice(s.Panel().Selected(m.activeNotices()))
}

// noticeDismissQueued reports whether the lane already carries a dismissal
// of id, running or waiting.
func (m *Model) noticeDismissQueued(id string) bool {
	var requests []uinotices.DismissRequest
	for _, job := range append([]*uieffects.Job{m.effects.Main.Active}, m.effects.Main.Pending...) {
		if job == nil {
			continue
		}
		if request, ok := job.Request.(uinotices.DismissRequest); ok {
			requests = append(requests, request)
		}
	}
	return uinotices.Queued(requests, id)
}

// applyNoticeDismiss restores a notice whose optimistic dismissal did not
// persist, and reopens the card only when the user has not moved on since.
func (m *Model) applyNoticeDismiss(job *uieffects.Job, err error) {
	if err == nil {
		return
	}
	failed := uinotices.FailedDismissal(job.Request.(uinotices.DismissRequest), err, m.effects.Quitting, m.gens.foreground, m.noticesOpen(), m.listClear())
	m.notices.Restore(failed.Restore)
	m.reportErr(failed.Err.Error())
	if failed.Reopen != "" {
		m.openNotices(failed.Reopen)
	}
}

// noticesServices is what the notices effect lane writes through.
func (m *Model) noticesServices() uinotices.Services {
	if m.services.store == nil {
		return uinotices.Services{}
	}
	return uinotices.Services{Store: m.services.store}
}

// statusRow is the status bar message as the panel's tail shows it, empty
// when the bar is.
func (m *Model) statusRow() string {
	if m.errBar.text == "" {
		return ""
	}
	return m.statusMessage(m.glyphs().ErroredGlyph, m.glyphs().FinishedGlyph, m.glyphs().WarningGlyph)
}

// noticesRoot is the root as the messages modal reads it.
type noticesRoot struct{ *Model }

func (r noticesRoot) ActiveNotices() []uinotices.Notice { return r.activeNotices() }

func (r noticesRoot) NoticesFrame() uinotices.Frame { return r.noticesFrame() }

func (r noticesRoot) MarkNotice(mark uinotices.Mark) {
	if mark.Dismiss {
		r.dismissNotice(mark.ID)
		return
	}
	r.seeNotice(mark.ID)
}

func (r noticesRoot) PaintNoticesCard(card []string, foot []presentation.PaintedKey) string {
	return r.paintNoticesCard(card, foot)
}

func (r noticesRoot) TakeForeground() { r.takeForeground() }

func (r noticesRoot) OpenLink(url string) tea.Cmd { return openLink(url) }

func (m *Model) routeNoticesMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case browserOpenMsg:
		m.handleBrowserOpen(msg)
		return routed(m, nil)
	case linkOpenErrMsg:
		m.reportErr(msg.err.Error())
		return routed(m, nil)
	case linkPageMsg:
		return routed(m, uiterminal.ShowLinkPage(msg.url))
	}
	return nil, nil, false
}
