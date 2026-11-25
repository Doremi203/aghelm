package ui

import (
	"slices"
	"time"

	uilink "github.com/Doremi203/aghelm/internal/ui/connections/link"
	uiapply "github.com/Doremi203/aghelm/internal/ui/dialogs/apply"
	uieffects "github.com/Doremi203/aghelm/internal/ui/effects"
	uiinstall "github.com/Doremi203/aghelm/internal/ui/launchhint/install"
	uiquicksend "github.com/Doremi203/aghelm/internal/ui/quick/send"
	uirailpersist "github.com/Doremi203/aghelm/internal/ui/rail/persist"
	uireviewsource "github.com/Doremi203/aghelm/internal/ui/review/source"
	uilifecycle "github.com/Doremi203/aghelm/internal/ui/session/lifecycle"
	uilaunch "github.com/Doremi203/aghelm/internal/ui/spawn/launch"
	tea "github.com/charmbracelet/bubbletea"
)

// effectState is the root's queue plus the fence a poll reads: a listing
// captured before the latest mutation finished is stale.
type effectState struct {
	uieffects.Queue
	latestObservation time.Time
}

func (m *Model) laneFor(request uieffects.Request) *uieffects.Lane {
	if _, remote := request.(uilink.Call); remote {
		return &m.effects.Remote
	}
	return &m.effects.Main
}

// effectStopWait bounds shutdown: an effect stuck on a hook or an unanswered
// prompt must not keep the process alive after its terminal is gone.
var effectStopWait = 5 * time.Second

// StopEffects refuses new effects and waits briefly for running ones. A quit
// the user forced, or work that outlives the wait, is recorded in
// AbandonedEffects.
func (m *Model) StopEffects() { m.effects.Stop(effectStopWait, effectName) }

// AbandonedEffects lists effects whose outcome the manager could not confirm
// before it exited.
func (m *Model) AbandonedEffects() []string { return slices.Clone(m.effects.Abandoned) }

func (m *Model) enqueueEffect(request uieffects.Request, chain uint64, first bool) {
	if m.effects.Quitting && !first {
		return
	}
	request = uieffects.Copy(request)
	m.effects.Push(m.laneFor(request), request, chain, first, m.bindEffect(request))
}

func (m *Model) nextEffectCmd() tea.Cmd { return m.effects.Next() }

func (m *Model) requestQuit() (tea.Model, tea.Cmd) {
	if install := m.ledger.install.Pending; install != nil {
		m.reportErr("install may still be running in " + install.Name + "; finish it, kill its terminal with the session controls, or attach and interrupt it before quitting")
		return m, nil
	}
	starting := func(job *uieffects.Job) (uiinstall.StartRequest, bool) {
		if job == nil {
			return uiinstall.StartRequest{}, false
		}
		request, ok := job.Request.(uiinstall.StartRequest)
		return request, ok
	}
	if request, ok := starting(m.effects.Main.Active); ok {
		m.reportErr("install for " + request.Binary + " is still starting; wait for its installer shell before quitting")
		return m, nil
	}
	for _, job := range m.effects.Main.Pending {
		if request, ok := starting(job); ok {
			m.reportErr("install for " + request.Binary + " is still queued; wait for its installer shell before quitting")
			return m, nil
		}
	}
	if m.effects.Quitting {
		return m, m.effects.Abandon(effectName)
	}
	m.effects.Quitting = true
	if awaited, ok := m.effects.FirstUnfinished(); ok {
		m.reportErr("finishing " + effectName(awaited.Request) + " before quitting; press the quit key again to leave without waiting")
	}
	return m, m.nextEffectCmd()
}

func (m *Model) handleEffectCompleted(msg uieffects.Completed) (tea.Model, tea.Cmd) {
	job := m.effects.Finish(msg.ID)
	if job == nil {
		return m, nil
	}
	switch job.Request.(type) {
	case uiquicksend.SendRequest, uiinstall.StartRequest, uilifecycle.Request, uilifecycle.AfterTurnRequest, uirailpersist.Request, uiapply.ForkEffect, uilaunch.Request, uilaunch.GroupRequest, uiapply.RenameEffect, uilifecycle.AckRequest, uilifecycle.AttachRequest:
		if msg.FinishedAt.After(m.effects.latestObservation) {
			m.effects.latestObservation = msg.FinishedAt
		}
	case uireviewsource.Change:
		// A send types into the agent pane, so a poll captured before it
		// is stale exactly like a lifecycle mutation; reads are not.
		if uireviewsource.IsSend(job.Request) && msg.FinishedAt.After(m.effects.latestObservation) {
			m.effects.latestObservation = msg.FinishedAt
		}
	}
	command := m.applyEffect(job, msg.Result, msg.Err)
	if m.poller != nil && pollAfterEffect(job.Request) {
		m.requestRefresh()
	}
	return m, tea.Batch(command, m.nextEffectCmd())
}

// pollAfterEffect reports whether a finished effect asks for a poll. The
// effects the poll itself enqueues every pass must not: the install settle
// check and the review status read would otherwise spin the poller for as
// long as the installer runs or the review stays open (#880).
func pollAfterEffect(request uieffects.Request) bool {
	switch request := request.(type) {
	case uiinstall.SettleRequest:
		return false
	case uireviewsource.Change:
		return request.Op != uireviewsource.Status
	}
	return true
}

func effectName(request uieffects.Request) string { return request.EffectName() }

func (m *Model) routeEffectMsg(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case uieffects.Completed:
		return routed(m.handleEffectCompleted(msg))
	}
	return nil, nil, false
}
