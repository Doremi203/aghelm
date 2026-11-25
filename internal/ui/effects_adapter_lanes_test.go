package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Doremi203/aghelm/internal/store"
	uidialogs "github.com/Doremi203/aghelm/internal/ui/dialogs"
	uirail "github.com/Doremi203/aghelm/internal/ui/rail"
	uilifecycle "github.com/Doremi203/aghelm/internal/ui/session/lifecycle"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSecondQuitAbandonsAStuckDrain(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.queueRail([]uirail.Mutation{{Kind: uirail.SaveCollapsed, Collapsed: []string{"a"}}, {Kind: uirail.SaveCollapsed, Collapsed: []string{"b"}}}, 0, false)
	if _, command := m.requestQuit(); command == nil || m.effects.Main.Active == nil {
		t.Fatal("first quit did not start draining")
	}
	_, command := m.requestQuit()
	if _, ok := command().(tea.QuitMsg); !ok {
		t.Fatal("second quit kept waiting on the drain")
	}
	want := []string{"rail change was still running", "rail change never started"}
	if got := m.AbandonedEffects(); !slices.Equal(got, want) {
		t.Fatalf("abandoned = %q, want %q", got, want)
	}
	if len(m.effects.Main.Pending) != 0 {
		t.Fatal("abandoned work stayed queued")
	}
	if !m.effects.Lifetime.Begin() {
		t.Fatal("lifetime closed early")
	}
	started := time.Now()
	m.StopEffects()
	if time.Since(started) > time.Second {
		t.Fatal("stop waited on work the user abandoned")
	}
}

func TestStopEffectsGivesUpOnStuckWork(t *testing.T) {
	m := buildModel(t)
	defer func(wait time.Duration) { effectStopWait = wait }(effectStopWait)
	effectStopWait = 10 * time.Millisecond
	m.queueRail([]uirail.Mutation{{Kind: uirail.SaveCollapsed, Collapsed: []string{"a"}}}, 0, false)
	m.nextEffectCmd()
	if !m.effects.Lifetime.Begin() {
		t.Fatal("lifetime refused work")
	}
	m.StopEffects()
	if got := m.AbandonedEffects(); !slices.Equal(got, []string{"rail change was still running"}) {
		t.Fatalf("abandoned = %q", got)
	}
}

// A quit that waits on accepted work says so, so a manager that does not
// close at once is not taken for a hang.
func TestQuitNamesTheWorkItWaitsFor(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.queueLifecycle(uilifecycle.Target{Action: uilifecycle.Action(uidialogs.ConfirmArchive), Sessions: []store.Session{{ID: "s"}}}, false, "")
	m.nextEffectCmd()
	m.requestQuit()
	if !strings.Contains(m.errBar.text, "finishing session archive") || !strings.Contains(m.errBar.text, "again") {
		t.Fatalf("status = %q, want the awaited archive and the second quit key named", m.errBar.text)
	}
	idle := buildModel(t)
	idle.requestQuit()
	if idle.errBar.text != "" {
		t.Fatalf("a quit with nothing pending reported %q", idle.errBar.text)
	}
}
