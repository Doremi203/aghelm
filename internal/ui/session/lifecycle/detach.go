package lifecycle

import (
	"github.com/Doremi203/aghelm/internal/tmux"
	uieffects "github.com/Doremi203/aghelm/internal/ui/effects"
	tea "github.com/charmbracelet/bubbletea"
)

// AckRequest clears the unread mark a finished session left.
type AckRequest struct{ SessionID string }

func (AckRequest) EffectName() string { return "acknowledge" }

func (r AckRequest) Run(s Services) (uieffects.Result, error) {
	return nil, s.Store.Acknowledge(r.SessionID)
}

// DetachRequest reads and clears the request a session wrote before it
// detached.
type DetachRequest struct{ SessionID string }

func (DetachRequest) EffectName() string { return "attach" }

func (r DetachRequest) Run(s Services) (uieffects.Result, error) {
	return Detached(s.Driver, r.SessionID)
}

type Detach struct {
	// Request is the marker the session wrote, empty when it wrote none.
	Request string
	// ClearErr is a failed clear, which leaves the marker for the next read.
	ClearErr error
}

// Detached reads and clears the request a session left before it detached.
func Detached(driver *tmux.Driver, sessionID string) (Detach, error) {
	marker, err := driver.PendingRequest(sessionID)
	if err != nil {
		return Detach{}, err
	}
	if marker == "" {
		return Detach{}, nil
	}
	return Detach{Request: marker, ClearErr: driver.ClearRequest(sessionID)}, nil
}

type DetachHost interface {
	ReportErr(text string)
	// Warn puts a caveat on the status bar, which a refused request is.
	Warn(text string)
	Refresh()
	Quitting() bool
	TakenSinceDetach() bool
	Name(sessionID string) string
	// Select moves the list's cursor to the session, false when it has no
	// row left.
	Select(sessionID string) bool
	Review(sessionID string) tea.Cmd
	Editor(sessionID string) tea.Cmd
}

// ApplyDetach opens the view a session asked for before it detached. The
// marker is already cleared, so a request the user moved past stays on the
// status bar instead of vanishing; moving the cursor is not moving past it.
func ApplyDetach(h DetachHost, request DetachRequest, result Detach, err error) tea.Cmd {
	if err != nil {
		h.ReportErr(err.Error())
		return nil
	}
	if result.ClearErr != nil {
		h.ReportErr(result.ClearErr.Error())
		h.Refresh()
		return nil
	}
	if h.Quitting() {
		return nil
	}
	if result.Request == "" {
		h.Refresh()
		return nil
	}
	if h.TakenSinceDetach() {
		h.Warn("the " + result.Request + " asked for in " + h.Name(request.SessionID) + " did not open: something else took the screen first; ask again")
		h.Refresh()
		return nil
	}
	if !h.Select(request.SessionID) {
		h.ReportErr("the session that asked for it has left the list")
		h.Refresh()
		return nil
	}
	switch result.Request {
	case tmux.RequestReview:
		return h.Review(request.SessionID)
	case tmux.RequestEditor:
		return h.Editor(request.SessionID)
	}
	h.Refresh()
	return nil
}
