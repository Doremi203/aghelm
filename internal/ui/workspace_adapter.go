package ui

import (
	"context"
	"errors"
	"time"

	"github.com/Doremi203/aghelm/internal/config"
	"github.com/Doremi203/aghelm/internal/execution"
	"github.com/Doremi203/aghelm/internal/git"
	"github.com/Doremi203/aghelm/internal/hooks"
	"github.com/Doremi203/aghelm/internal/notify"
	"github.com/Doremi203/aghelm/internal/sessioncmd"
	"github.com/Doremi203/aghelm/internal/status"
	"github.com/Doremi203/aghelm/internal/store"
	"github.com/Doremi203/aghelm/internal/sysstat"
	"github.com/Doremi203/aghelm/internal/tmux"
	uiworkspace "github.com/Doremi203/aghelm/internal/ui/workspace"
	tea "github.com/charmbracelet/bubbletea"
)

type workspace struct {
	sessions []store.Session
	// tmuxSocket is the server the last poll read panes from; rows stamped
	// with another server belong to a manager running against it, and rows
	// stamped by none belong to whichever manager holds the store.
	tmuxSocket     string
	leadingManager bool
	groups         []string
	groupPaths     map[string]string
	groupWorktrees map[string]string
	groupBases     map[string]string
	// groupEnvs is each group's own environment profile, envProfiles the
	// name of every defined one, and envKeys the variables each sets.
	groupEnvs      map[string]string
	envProfiles    []string
	envKeys        map[string][]string
	archivedGroups map[string]bool
	snap           sysstat.Snapshot
	agents         agentStats
	// queuedMessages is replaced whole on every refresh rather than merged,
	// so a delivered message's badge clears itself.
	queuedMessages map[string]int
	// paneLines holds each session's last meaningful pane line, which the
	// full screen row's second line quotes. Merged rather than replaced, so
	// a session that lost its window keeps its last words.
	paneLines map[string]string
	// panePrompts holds the last prompt each session's transcript echoes,
	// which the full screen row shows as the task it is on.
	panePrompts map[string]string
	// panes is the last pass's agent pane geometry, which the poller reads
	// off the UI loop alongside its liveness listing.
	panes map[string]tmux.Pane
	net   uiworkspace.Net
	// lastListedAt is the newest listing applied; an older one still carries
	// a notification's selection, turn ends and pane facts, but not the rows
	// it saw.
	lastListedAt time.Time
}

func (m *Model) railFootLines(width int) []string {
	if m.prefs.hideStats {
		return nil
	}
	return uiworkspace.FootLine(m.workspace.snap, m.workspace.net, width, m.prefs.glyphMode)
}

var postNotification = notify.Notify

var showCommand = notify.ShowCommand

type poller struct {
	runner       *execution.Runner
	dependencies execution.Dependencies
	options      execution.Options
}

func newPoller(st *store.Store, driver *tmux.Driver, engine *status.Engine, hookManager *hooks.Manager, gitDriver *git.Driver, statusSources, sessionStores, mcpStyles, reportStyles map[string]string, shellTools map[string]bool, binaries execution.ToolBinaries, interval time.Duration) *poller {
	deps := execution.Dependencies{Store: st, TMux: driver, Engine: engine, Hooks: hookManager, Git: gitDriver, Notify: postNotification, ShowCommand: showCommand, TakeFocus: takeNotifyFocus}
	opts := execution.Options{StatusSources: statusSources, SessionStores: sessionStores, MCPStyles: mcpStyles, ReportStyles: reportStyles, ShellTools: shellTools, Binaries: binaries, Interval: interval}
	return &poller{runner: execution.New(deps, opts), dependencies: deps, options: opts}
}

func pollMessage(result execution.Result) tea.Msg {
	if result.Err != nil {
		return errMsg{result.Err}
	}
	s := result.Snapshot
	return refreshMsg{sessions: s.Sessions, listedAt: s.ListedAt, groups: s.Groups, groupPaths: s.GroupPaths,
		groupWorktrees: s.GroupWorktrees, groupBases: s.GroupBases, groupEnvs: s.GroupEnvs, envProfiles: s.EnvProfiles, envKeys: s.EnvKeys, archivedGroups: s.ArchivedGroups, snap: s.Snap, snapOK: s.SnapOK,
		agents:         agentStats{count: s.Agents.Count, cpu: s.Agents.CPU, ram: s.Agents.RAM, rss: s.Agents.RSS},
		queuedMessages: s.QueuedMessages, paneLines: s.PaneLines, panePrompts: s.PanePrompts, panes: s.Panes,
		tmuxSocket: s.TmuxSocket, leadingManager: s.LeadingManager, focusID: s.FocusID, turnsEnded: s.TurnsEnded, followWarning: s.FollowWarning}
}

func (p *poller) refreshOnce() tea.Msg                                { return pollMessage(p.runner.Step()) }
func (p *poller) setInput(archived bool)                              { p.runner.SetInput(archived) }
func (p *poller) requestRefresh()                                     { p.runner.RequestRefresh() }
func (p *poller) results(ctx context.Context) <-chan execution.Result { return p.runner.Run(ctx) }

func takeNotifyFocus() (string, bool) {
	dir, err := config.Dir()
	if err != nil {
		return "", false
	}
	return notify.TakeFocus(dir)
}

const quoteHistoryLines = 300

func ignoreDeletedSession(err error) error {
	if errors.Is(err, store.ErrSessionGone) {
		return nil
	}
	return err
}

// agentStats aggregates process-tree usage across all live sessions.
// cpu and ram are shares of this machine (0–100); rss is absolute bytes.
type agentStats struct {
	count int
	cpu   float64
	ram   float64
	rss   uint64
}

type refreshMsg struct {
	sessions []store.Session
	// listedAt is when the pass read that list, which is a whole pass of
	// tmux and ps calls before the UI sees it.
	listedAt       time.Time
	groups         []string
	groupPaths     map[string]string
	groupWorktrees map[string]string
	groupBases     map[string]string
	groupEnvs      map[string]string
	envProfiles    []string
	envKeys        map[string][]string
	archivedGroups map[string]bool
	snap           sysstat.Snapshot
	snapOK         bool
	agents         agentStats
	queuedMessages map[string]int
	paneLines      map[string]string
	panePrompts    map[string]string
	// panes is the pass's agent pane geometry, read off the UI loop with
	// the liveness listing the poller already makes.
	panes map[string]tmux.Pane
	// tmuxSocket is the server this pass read panes from, which tells the
	// rows apart from ones another manager's server holds.
	tmuxSocket string
	// leadingManager is whether this pass held the store, which decides
	// whether the rows no server has claimed are this manager's to show as
	// current.
	leadingManager bool
	// focusID is the session a clicked notification named, taken from
	// the config directory by this pass.
	focusID string
	// turnsEnded are the sessions whose turn this pass saw end after they
	// asked to be archived or killed once it did.
	turnsEnded []string
	// followWarning reports a session whose conversation the pass could not
	// follow, the first time it fails that way.
	followWarning string
}

// StartPoller launches the background polling loop. It runs outside the
// bubbletea event loop so statuses keep updating while the TUI is
// suspended inside a tmux attach.
func (m *Model) StartPoller(ctx context.Context, send func(tea.Msg)) <-chan struct{} {
	m.syncPollInput()
	results := m.poller.results(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for result := range results {
			send(pollMessage(result))
		}
	}()
	return done
}

// syncPollInput keeps the loaded projection complete in both views. Rail
// policy decides which archive state to render, while confirmation cards
// need hidden children too and must not read SQLite on the Update path.
func (m *Model) syncPollInput() {
	m.poller.setInput(true)
}

// requestRefresh publishes the current UI state to the poller and asks
// for an immediate pass.
func (m *Model) requestRefresh() {
	m.syncPollInput()
	m.poller.requestRefresh()
}

// refreshExistingSessionUX re-applies the tmux bindings and status bar to
// sessions that were already running when the manager started, so a session
// created before an update still gets the current key bindings (the
// server-global Ctrl+R review key) and footer.
func (m *Model) refreshExistingSessionUX() tea.Msg {
	if err := m.services.tmux.EnsureBindings(); err != nil {
		return errMsg{err}
	}
	sessions, err := m.services.store.ListSessions(true)
	if err != nil {
		return errMsg{err}
	}
	for _, sess := range sessions {
		if !m.services.tmux.Exists(sess.ID) {
			continue
		}
		// Best-effort per session: one that dies between the check and here
		// errors harmlessly and must not abort the rest, and the bindings that
		// matter are already installed above.
		_ = m.services.tmux.RefreshChrome(sess.ID)
		_ = m.services.tmux.SetLabel(sess.ID, sessioncmd.SessionLabel(sess.Group, sess.Name))
	}
	return nil
}

// refreshCmd runs one synchronous polling pass; the background poller
// covers normal operation, this exists for tests and explicit refreshes.
func (m *Model) refreshCmd() tea.Cmd {
	m.syncPollInput()
	return func() tea.Msg {
		return m.poller.refreshOnce()
	}
}

// listingFenced reports a listing no newer than the last committed mutation,
// whose rows would show the store as it was before that mutation.
func (m *Model) listingFenced(listedAt time.Time) bool {
	return !listedAt.IsZero() && !m.effects.latestObservation.IsZero() && !listedAt.After(m.effects.latestObservation)
}

func (m *Model) routeRefreshMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case refreshMsg:
		// A listing older than a committed mutation or a newer listing keeps
		// its rows out, but still carries a selection from a notification,
		// the turns it saw end and its pane facts.
		fenced := m.listingFenced(msg.listedAt)
		staleListing := fenced || (!msg.listedAt.IsZero() && msg.listedAt.Before(m.workspace.lastListedAt))
		if !staleListing && !msg.listedAt.IsZero() {
			m.workspace.lastListedAt = msg.listedAt
		}
		if fenced {
			m.requestRefresh()
		}
		m.startup.Booting = false
		m.ageError()
		if msg.followWarning != "" {
			m.reportWarn(msg.followWarning)
		}
		if !staleListing {
			sessions := m.ledger.lifecycle.DropRecentlyRemoved(m.ledger.lifecycle.KeepPendingLaunches(msg.sessions, m.workspace.sessions, msg.listedAt), msg.listedAt)
			m.reconcileListing(&msg)
			m.workspace.sessions = sessions
			m.workspace.groups = msg.groups
			m.workspace.groupPaths = msg.groupPaths
			m.workspace.groupWorktrees = msg.groupWorktrees
			m.workspace.groupBases = msg.groupBases
			m.workspace.groupEnvs = msg.groupEnvs
			m.workspace.envProfiles = msg.envProfiles
			m.workspace.envKeys = msg.envKeys
			if f := m.form(); f != nil && f.Remote.Host == "" {
				f.Env().SetProfiles(msg.envProfiles)
			}
			if q := m.quick(); q != nil {
				q.Env().SetProfiles(msg.envProfiles)
			}
			m.workspace.archivedGroups = msg.archivedGroups
		}
		m.endAfterTurns(msg.turnsEnded)
		m.workspace.tmuxSocket = msg.tmuxSocket
		m.workspace.leadingManager = msg.leadingManager
		m.workspace.panes = msg.panes
		m.workspace.agents = msg.agents
		m.workspace.queuedMessages = msg.queuedMessages
		if m.workspace.paneLines == nil {
			m.workspace.paneLines = map[string]string{}
		}
		for id, line := range msg.paneLines {
			m.workspace.paneLines[id] = line
		}
		if m.workspace.panePrompts == nil {
			m.workspace.panePrompts = map[string]string{}
		}
		for id, prompt := range msg.panePrompts {
			if prompt != "" {
				m.workspace.panePrompts[id] = prompt
			}
		}
		if msg.snapOK {
			m.workspace.snap = msg.snap
			m.workspace.net.Update(msg.snap.NetSent, msg.snap.NetRecv, msg.snap.NetOK, time.Now())
		}
		// Sessions left from a previous run carry that run's window size,
		// which the cache knows nothing about; the geometry cache adopts their
		// real geometry on the first pass, so nothing resets the cache here.
		if !m.startup.SessionsSized && m.layout.width > 0 && len(m.workspace.sessions) > 0 {
			m.startup.SessionsSized = true
		}
		m.publishPaneSize()
		// The body changes height for more reasons than a terminal resize:
		// the quick bar opening, the status line appearing, a new badge in
		// the header. Every pass grows what falls short; the call is free
		// when nothing moved, since it diffs against the cached sizes.
		if m.startup.SessionsSized && m.layout.width > 0 {
			m.resizeSessions()
		}
		m.settleInstall()
		m.rebuildRows()
		if msg.focusID != "" {
			m.focusSession(msg.focusID)
		}
		return routed(m, tea.Batch(m.diffRefreshCmd(), m.reviewStatusesCmd(), m.startStartupTick(), m.ssh.Poll()))
	}
	return nil, nil, false
}
