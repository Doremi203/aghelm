package panes

import uieffects "github.com/Doremi203/aghelm/internal/ui/effects"

// Services is what the geometry effects run against: the store the pane box
// is written to, the tmux server a resize pins, and the poller's run mutex.
type Services struct {
	Store    Store
	Driver   Driver
	Reflower Reflower
}

// ResizeRequest is the effect lane's pane resize: this package's Resize and
// nothing else.
type ResizeRequest struct{ Resize }

func (ResizeRequest) EffectName() string { return "pane resize" }

func (r ResizeRequest) EffectCopy() uieffects.Request {
	r.Resize = Resize{Publish: r.Publish, Targets: append([]PaneResize(nil), r.Targets...)}
	return r
}

func (r ResizeRequest) Run(s Services) (uieffects.Result, error) {
	return Run(s.Store, s.Driver, s.Reflower, r.Resize)
}
