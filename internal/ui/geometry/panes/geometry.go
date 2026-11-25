// Package panes is the sessions' geometry runtime: it sizes every live
// session's tmux window to the body box, remembers the size it last pinned,
// and publishes that box for the launch paths that run without a manager.
package panes

import "github.com/Doremi203/aghelm/internal/tmux"

type PaneResize struct {
	ID   string
	Size [2]int
}

// Resize is one queued resize: a box to publish for the launch paths that
// run without a manager, and the panes to pin to it.
type Resize struct {
	Publish [2]int
	Targets []PaneResize
}

// Empty reports whether the resize would do nothing.
func (r Resize) Empty() bool { return len(r.Targets) == 0 && r.Publish == [2]int{} }

type Session struct {
	ID        string
	Tool      string
	AltScreen bool
	Archived  bool
}

// Box is where sessions are pinned: the whole terminal body.
type Box struct {
	Width  int
	Height int
}

// Result is what a resize landed: the panes tmux accepted and the box the
// store took.
type Result struct {
	Publish [2]int
	Resized []PaneResize
}

// Panes is what this feature remembers between passes: the box each pane was
// last pinned to, the pane processes behind those sizes, and the box last
// published.
type Panes struct {
	// LastSizes is the box each pane was last pinned to; [0 0] marks a pin
	// still owed to a session whose window this run just created.
	LastSizes map[string][2]int
	// LastPIDs is each pane's process when its size was measured, so a
	// revive outside this process reads as a replacement, not as drift.
	LastPIDs map[string]int
	// Published is the box last written for the launch paths that run
	// without a manager.
	Published [2]int
}

type Store interface {
	SetPaneSize(width, height int) error
}

type Driver interface {
	Resize(id string, width, height int) error
}

// Reflower runs work with the poller's run mutex held, so a size that lands
// mid-pass is not read as the agent repainting.
type Reflower interface {
	ReflowSessions(ids []string, reflow func()) error
}

// Root is the root as this runtime reaches it: what a shrink costs a tool,
// and the effect lane a pin rides.
type Root interface {
	// FitsHeight reports whether the tool repaints its own screen, so a
	// shorter box cannot cost it scrollback.
	FitsHeight(tool string) bool
	Queue(request Resize)
}

// Apply records what a resize landed, so the next pass diffs against it.
func (p *Panes) Apply(result Result) {
	if p.LastSizes == nil {
		p.LastSizes = map[string][2]int{}
	}
	for _, target := range result.Resized {
		p.LastSizes[target.ID] = target.Size
	}
	if result.Publish != [2]int{} {
		p.Published = result.Publish
	}
}

// QueueAll pins every session whose pane is not already at the box. Width
// changes and height growth pin eagerly; a box that lost rows, to transient
// chrome or a shorter terminal, leaves the pane tall and lets the view crop
// instead, because a height shrink makes Codex clear the pane's entire
// scrollback (#369). A pane on the alternate screen holds no scrollback for a
// shrink to clear, and a tool with fits_height keeps its own, so those follow
// the box down and a full-screen TUI keeps its top rows in view.
func (p *Panes) QueueAll(root Root, box Box, sessions []Session, panes map[string]tmux.Pane) {
	if box.Width <= 0 || box.Height <= 0 {
		return
	}
	if p.LastSizes == nil {
		p.LastSizes = map[string][2]int{}
	}
	p.MarkReplacedFresh(panes)
	p.Seed(panes)
	var targets []PaneResize
	for _, sess := range sessions {
		if sess.Archived {
			continue
		}
		// A dead row has no window to size. Its resize would fail every
		// pass, and a finished effect asks for another poll, so the rows
		// without a pane in this pass's listing are skipped (#880).
		if _, live := panes[sess.ID]; !live {
			continue
		}
		wanted := box.Height
		if last, ok := p.LastSizes[sess.ID]; ok {
			fitsHeight := sess.AltScreen || root.FitsHeight(sess.Tool)
			keepsHeight := last[1] > box.Height && !fitsHeight
			if last[0] == box.Width && (last[1] == box.Height || keepsHeight) {
				continue
			}
			// A width re-pin of a taller pane keeps its height: shrinking it
			// would clear a Codex scrollback (#369); the painted view crops.
			if keepsHeight {
				wanted = last[1]
			}
		}
		targets = append(targets, PaneResize{ID: sess.ID, Size: [2]int{box.Width, wanted}})
	}
	if len(targets) == 0 {
		return
	}
	root.Queue(Resize{Targets: targets})
}

// MarkFresh queues one exact size pin for a session whose window this run
// just created. Its launch size came from pre-selection geometry, and with
// nothing in its scrollback yet the one re-pin is free, unlike the panes
// adopted from a previous run, which Seed protects.
func (p *Panes) MarkFresh(id string) {
	if p.LastSizes == nil {
		p.LastSizes = map[string][2]int{}
	}
	p.LastSizes[id] = [2]int{0, 0}
}

// Publish records the box for the launch paths that run without a manager:
// the CLI and the MCP server open a pane with nothing to ask for the body
// geometry, and tmux gives an unsized detached session 80x24.
func (p *Panes) Publish(root Root, width, height int) {
	if width <= 0 || height <= 0 || p.Published == [2]int{width, height} {
		return
	}
	root.Queue(Resize{Publish: [2]int{width, height}})
}

// Forget drops a session's cached size, so its next pin is exact.
func (p *Panes) Forget(id string) {
	delete(p.LastSizes, id)
}

// MarkReplacedFresh spots a session whose pane process changed since its
// geometry was measured: a revive outside this process put the row on a
// window the manager never sized, and the cached size of the window it
// replaced would otherwise read as already matching the box, leaving the new
// pane at whatever it was born with for the life of the run. The replacement
// has no scrollback to protect, so it takes MarkFresh's exact pin rather than
// the adopted-pane treatment that keeps a taller height.
func (p *Panes) MarkReplacedFresh(panes map[string]tmux.Pane) {
	if p.LastPIDs == nil {
		p.LastPIDs = map[string]int{}
	}
	for id, pane := range panes {
		last := p.LastPIDs[id]
		p.LastPIDs[id] = pane.PID
		if last != 0 && last != pane.PID {
			p.MarkFresh(id)
		}
	}
}

// Seed fills the geometry cache from the pass's pane listing for sessions
// this run has not sized yet, so a pane adopted from a previous run is not
// shrunk (and its Codex scrollback cleared) just to match a box it already
// exceeds. A split window is taken every pass instead: the agent opened those
// panes out of its own window, taking room from the agent's own pane,
// and no manager action precedes that for the cache to follow. Drift the
// manager did not cause otherwise stays, so a window someone resized from
// another client is left where they put it.
func (p *Panes) Seed(panes map[string]tmux.Pane) {
	for id, geom := range panes {
		last, sized := p.LastSizes[id]
		// MarkFresh's pin is still owed: a session created this run carries
		// its pre-selection launch size, not the box.
		if sized && last == [2]int{0, 0} {
			continue
		}
		if sized && geom.Panes < 2 {
			continue
		}
		p.LastSizes[id] = [2]int{geom.Width, geom.Height}
	}
}

// Run executes a queued resize: the box is recorded first, so a launch racing
// it still reads the size the manager is about to paint.
func Run(st Store, driver Driver, reflower Reflower, request Resize) (Result, error) {
	result := Result{}
	if request.Publish != [2]int{} {
		if err := st.SetPaneSize(request.Publish[0], request.Publish[1]); err != nil {
			return result, err
		}
		result.Publish = request.Publish
	}
	if len(request.Targets) == 0 {
		return result, nil
	}
	ids := make([]string, len(request.Targets))
	for i, target := range request.Targets {
		ids[i] = target.ID
	}
	pin := func() { result.Resized = resizeAll(driver, request.Targets) }
	if reflower == nil {
		pin()
		return result, nil
	}
	if err := reflower.ReflowSessions(ids, pin); err != nil {
		return result, err
	}
	return result, nil
}

// resizeAll pins every target in parallel, so a fleet of sessions does not
// serialize N tmux round trips on the UI path.
func resizeAll(driver Driver, targets []PaneResize) []PaneResize {
	results := make(chan PaneResize, len(targets))
	for _, target := range targets {
		go func(target PaneResize) {
			if driver.Resize(target.ID, target.Size[0], target.Size[1]) != nil {
				target.ID = ""
			}
			results <- target
		}(target)
	}
	resized := make([]PaneResize, 0, len(targets))
	for range targets {
		if target := <-results; target.ID != "" {
			resized = append(resized, target)
		}
	}
	return resized
}

// Effective drops the targets and the box the queued resizes already carry,
// so a burst of window sizes queues one pin per pane at the newest size. A
// newer size for the same pane still queues.
func Effective(request Resize, queued []Resize) Resize {
	desired := map[string][2]int{}
	published := [2]int{}
	for _, queued := range queued {
		for _, target := range queued.Targets {
			desired[target.ID] = target.Size
		}
		if queued.Publish != [2]int{} {
			published = queued.Publish
		}
	}
	targets := make([]PaneResize, 0, len(request.Targets))
	for _, target := range request.Targets {
		if desired[target.ID] != target.Size {
			targets = append(targets, target)
		}
	}
	request.Targets = targets
	if published == request.Publish {
		request.Publish = [2]int{}
	}
	return request
}

// Merge folds request into the geometry job queued just before it, which is
// the only job a window size may coalesce with: folding across an accepted
// mutation would reorder it past that mutation.
func Merge(previous, request Resize) Resize {
	merged := Resize{Publish: previous.Publish, Targets: append([]PaneResize(nil), previous.Targets...)}
	byID := map[string]int{}
	for i, target := range merged.Targets {
		byID[target.ID] = i
	}
	for _, target := range request.Targets {
		if i, ok := byID[target.ID]; ok {
			merged.Targets[i] = target
			continue
		}
		byID[target.ID] = len(merged.Targets)
		merged.Targets = append(merged.Targets, target)
	}
	if request.Publish != [2]int{} {
		merged.Publish = request.Publish
	}
	return merged
}
