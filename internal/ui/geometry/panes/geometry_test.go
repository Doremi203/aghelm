package panes

import (
	"errors"
	"testing"

	"github.com/Doremi203/aghelm/internal/tmux"
)

type fakeStore struct {
	panes   [][2]int
	paneErr error
}

func (s *fakeStore) SetPaneSize(width, height int) error {
	if s.paneErr != nil {
		return s.paneErr
	}
	s.panes = append(s.panes, [2]int{width, height})
	return nil
}

type fakeDriver struct {
	sizes map[string][2]int
	fails map[string]bool
}

func (d *fakeDriver) Resize(id string, width, height int) error {
	if d.fails[id] {
		return errors.New("no such session")
	}
	if d.sizes == nil {
		d.sizes = map[string][2]int{}
	}
	d.sizes[id] = [2]int{width, height}
	return nil
}

type fakeRoot struct {
	fits   map[string]bool
	queued []Resize
}

func (r *fakeRoot) FitsHeight(tool string) bool { return r.fits[tool] }
func (r *fakeRoot) Queue(request Resize)        { r.queued = append(r.queued, request) }

func TestApplyRecordsTheBoxAPassLanded(t *testing.T) {
	p := &Panes{}
	p.Apply(Result{Publish: [2]int{80, 20}, Resized: []PaneResize{{ID: "a", Size: [2]int{40, 20}}}})
	if p.LastSizes["a"] != [2]int{40, 20} || p.Published != [2]int{80, 20} {
		t.Fatalf("cache = %v / %v", p.LastSizes, p.Published)
	}
	p.Apply(Result{})
	if p.LastSizes["a"] != [2]int{40, 20} || p.Published != [2]int{80, 20} {
		t.Fatalf("an empty result cleared the cache: %v / %v", p.LastSizes, p.Published)
	}
}

func TestQueueAllPinsOnlyWhatMoved(t *testing.T) {
	root := &fakeRoot{fits: map[string]bool{}}
	p := &Panes{}
	sessions := []Session{{ID: "a", Tool: "codex"}, {ID: "b", Tool: "claude"}, {ID: "gone", Archived: true}}
	panes := map[string]tmux.Pane{"a": {PID: 1}, "b": {PID: 2, AltScreen: true}}

	p.QueueAll(root, Box{Width: 40, Height: 20}, sessions, panes)
	if len(root.queued) != 1 || len(root.queued[0].Targets) != 2 {
		t.Fatalf("first pass queued %+v, want both live sessions", root.queued)
	}
	root.queued = nil
	p.Apply(Result{Resized: []PaneResize{{ID: "a", Size: [2]int{40, 20}}, {ID: "b", Size: [2]int{40, 20}}}})

	p.QueueAll(root, Box{Width: 40, Height: 20}, sessions, panes)
	if len(root.queued) != 0 {
		t.Fatalf("an unchanged box queued %+v", root.queued)
	}
}

func TestQueueAllKeepsATallerPaneForScrolledTools(t *testing.T) {
	root := &fakeRoot{fits: map[string]bool{}}
	p := &Panes{LastSizes: map[string][2]int{"a": {40, 30}, "b": {40, 30}}}
	panes := map[string]tmux.Pane{"a": {PID: 1}, "b": {PID: 2, AltScreen: true}}

	p.QueueAll(root, Box{Width: 40, Height: 20}, []Session{{ID: "a", Tool: "codex"}, {ID: "b", Tool: "codex", AltScreen: true}}, panes)
	if len(root.queued) != 1 {
		t.Fatalf("queued %+v, want the alternate-screen pane", root.queued)
	}
	targets := root.queued[0].Targets
	if len(targets) != 1 || targets[0].ID != "b" || targets[0].Size != [2]int{40, 20} {
		t.Fatalf("targets = %+v, want b at the box", targets)
	}
}

func TestQueueAllFollowsTheBoxDownForFittingTools(t *testing.T) {
	root := &fakeRoot{fits: map[string]bool{"claude": true}}
	p := &Panes{LastSizes: map[string][2]int{"a": {40, 30}}}
	panes := map[string]tmux.Pane{"a": {PID: 1}}

	p.QueueAll(root, Box{Width: 40, Height: 20}, []Session{{ID: "a", Tool: "claude"}}, panes)
	if len(root.queued) != 1 || root.queued[0].Targets[0].Size != [2]int{40, 20} {
		t.Fatalf("queued %+v, want the fitting tool down to the box", root.queued)
	}
}

func TestMarkFreshOWesAnExactPinAndSeedLeavesItOwed(t *testing.T) {
	p := &Panes{}
	p.MarkFresh("a")
	if p.LastSizes["a"] != [2]int{0, 0} {
		t.Fatalf("fresh pane cached %v", p.LastSizes["a"])
	}
	p.Seed(map[string]tmux.Pane{"a": {Width: 200, Height: 50, PID: 9}})
	if p.LastSizes["a"] != [2]int{0, 0} {
		t.Fatalf("seed adopted a window the manager never sized: %v", p.LastSizes["a"])
	}
}

func TestSeedAdoptsUnsizedPanesAndSplitWindows(t *testing.T) {
	p := &Panes{LastSizes: map[string][2]int{"adopted": {0, 0}}}
	delete(p.LastSizes, "adopted")
	p.Seed(map[string]tmux.Pane{
		"adopted": {Width: 200, Height: 50, PID: 1},
		"split":   {Width: 90, Height: 30, Panes: 2, PID: 2},
	})
	if p.LastSizes["adopted"] != [2]int{200, 50} || p.LastSizes["split"] != [2]int{90, 30} {
		t.Fatalf("cache = %v", p.LastSizes)
	}

	sized := &Panes{LastSizes: map[string][2]int{"kept": {40, 20}}}
	sized.Seed(map[string]tmux.Pane{"kept": {Width: 120, Height: 40, PID: 3}})
	if sized.LastSizes["kept"] != [2]int{40, 20} {
		t.Fatalf("drift the manager did not cause was taken: %v", sized.LastSizes["kept"])
	}
}

func TestMarkReplacedFreshSpotsAJobChange(t *testing.T) {
	p := &Panes{LastSizes: map[string][2]int{"a": {40, 20}, "b": {40, 20}}, LastPIDs: map[string]int{"a": 7, "b": 8}}
	p.MarkReplacedFresh(map[string]tmux.Pane{"a": {PID: 7}, "b": {PID: 9}})
	if p.LastSizes["a"] != [2]int{40, 20} {
		t.Fatalf("an unchanged pane lost its size: %v", p.LastSizes["a"])
	}
	if p.LastSizes["b"] != [2]int{0, 0} {
		t.Fatalf("a replaced pane kept the old window's size: %v", p.LastSizes["b"])
	}
}

func TestForgetDropsTheCachedSize(t *testing.T) {
	p := &Panes{LastSizes: map[string][2]int{"a": {40, 20}}}
	p.Forget("a")
	if _, ok := p.LastSizes["a"]; ok {
		t.Fatal("forget left the cache entry")
	}
}

func TestPublishRecordsTheBoxOnce(t *testing.T) {
	root := &fakeRoot{}
	p := &Panes{}
	p.Publish(root, 0, 20)
	p.Publish(root, 80, 20)
	p.Apply(Result{Publish: [2]int{80, 20}})
	p.Publish(root, 80, 20)
	if len(root.queued) != 1 || root.queued[0].Publish != [2]int{80, 20} {
		t.Fatalf("queued %+v, want one publish", root.queued)
	}
}

func TestRunPublishesThenPinsThroughTheReflow(t *testing.T) {
	store := &fakeStore{}
	driver := &fakeDriver{fails: map[string]bool{"b": true}}
	result, err := Run(store, driver, nil, Resize{
		Publish: [2]int{80, 20},
		Targets: []PaneResize{{ID: "a", Size: [2]int{40, 20}}, {ID: "b", Size: [2]int{40, 20}}},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(store.panes) != 1 || store.panes[0] != [2]int{80, 20} {
		t.Fatalf("published %+v", store.panes)
	}
	if result.Publish != [2]int{80, 20} || len(result.Resized) != 1 || result.Resized[0].ID != "a" {
		t.Fatalf("result = %+v", result)
	}
}

func TestRunLeavesThePanesAloneWhenTheStoreRefusesTheBox(t *testing.T) {
	driver := &fakeDriver{}
	_, err := Run(&fakeStore{paneErr: errors.New("locked")}, driver, nil, Resize{
		Publish: [2]int{80, 20},
		Targets: []PaneResize{{ID: "a", Size: [2]int{40, 20}}},
	})
	if err == nil {
		t.Fatal("a refused box should fail the run")
	}
	if len(driver.sizes) != 0 {
		t.Fatalf("panes were pinned anyway: %v", driver.sizes)
	}
}

type fakeReflower struct {
	ids  []string
	err  error
	runs int
}

func (r *fakeReflower) ReflowSessions(ids []string, reflow func()) error {
	r.ids, r.runs = ids, r.runs+1
	if r.err != nil {
		return r.err
	}
	reflow()
	return nil
}

func TestRunHoldsThePollMutexWhileItPins(t *testing.T) {
	reflower := &fakeReflower{}
	if _, err := Run(&fakeStore{}, &fakeDriver{}, reflower, Resize{Targets: []PaneResize{{ID: "a", Size: [2]int{40, 20}}}}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if reflower.runs != 1 || len(reflower.ids) != 1 || reflower.ids[0] != "a" {
		t.Fatalf("reflow = %+v", reflower)
	}
}

func TestEffectiveDropsWhatIsAlreadyQueued(t *testing.T) {
	queued := []Resize{{Publish: [2]int{80, 20}, Targets: []PaneResize{{ID: "a", Size: [2]int{40, 20}}}}}
	request := Effective(Resize{Publish: [2]int{80, 20}, Targets: []PaneResize{
		{ID: "a", Size: [2]int{40, 20}},
		{ID: "b", Size: [2]int{40, 20}},
	}}, queued)
	if request.Publish != [2]int{} || len(request.Targets) != 1 || request.Targets[0].ID != "b" {
		t.Fatalf("effective = %+v", request)
	}
	if !Effective(Resize{Targets: []PaneResize{{ID: "a", Size: [2]int{40, 20}}}}, queued).Empty() {
		t.Fatal("an already queued target was not dropped")
	}
}

func TestMergeFoldsTargetsAndTakesTheNewerBox(t *testing.T) {
	previous := Resize{Publish: [2]int{80, 20}, Targets: []PaneResize{{ID: "a", Size: [2]int{40, 20}}}}
	merged := Merge(previous, Resize{Publish: [2]int{90, 25}, Targets: []PaneResize{
		{ID: "a", Size: [2]int{40, 25}},
		{ID: "b", Size: [2]int{40, 25}},
	}})
	if len(merged.Targets) != 2 || merged.Targets[0].ID != "a" || merged.Targets[0].Size != [2]int{40, 25} {
		t.Fatalf("merged targets = %+v", merged.Targets)
	}
	if merged.Publish != [2]int{90, 25} {
		t.Fatalf("merged publish = %v", merged.Publish)
	}
	if len(previous.Targets) != 1 || previous.Targets[0].Size != [2]int{40, 20} {
		t.Fatalf("merge mutated the queued request: %+v", previous.Targets)
	}
}
