package ui

import (
	"fmt"
	"go/ast"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	uilifecycle "github.com/Doremi203/aghelm/internal/ui/session/lifecycle"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"

	"github.com/Doremi203/aghelm/internal/keybind"
	"github.com/Doremi203/aghelm/internal/remote"
	"github.com/Doremi203/aghelm/internal/store"
	uidialogs "github.com/Doremi203/aghelm/internal/ui/dialogs"
	uihelp "github.com/Doremi203/aghelm/internal/ui/help"
	uiinstall "github.com/Doremi203/aghelm/internal/ui/launchhint/install"
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uireview "github.com/Doremi203/aghelm/internal/ui/review"
	uisettings "github.com/Doremi203/aghelm/internal/ui/settings"
	uisurface "github.com/Doremi203/aghelm/internal/ui/surface"
	uiterminal "github.com/Doremi203/aghelm/internal/ui/terminal"
	tea "github.com/charmbracelet/bubbletea"
)

// A bare Model must default to mouse reporting on: mouseDisabled is named
// for its off polarity precisely so a zero-valued struct, the shape most
// tests build, does not silently disable the mouse everywhere.
func TestSyncMouseCaptureDefaultsOn(t *testing.T) {
	m := &Model{}
	// Nothing has changed from the zero value's implied "captured" state,
	// so there is nothing to tell the terminal.
	if cmd := m.syncMouseCapture(); cmd != nil {
		t.Fatal("default state should not re-announce mouse capture")
	}
	if m.mouse.released {
		t.Fatal("mouse should stay captured by default")
	}
}

// The mouse-mode setting releases the mouse everywhere except focus mode,
// whose own forwarding predates the setting and stays on regardless.
func TestSyncMouseCaptureRespectsMouseDisabled(t *testing.T) {
	m := &Model{prefs: preferences{mouseDisabled: true}}
	if cmd := m.syncMouseCapture(); cmd == nil {
		t.Fatal("turning the setting off should release the mouse")
	}
	if !m.mouse.released {
		t.Fatal("mouseReleased should follow the setting in list mode")
	}
}

// paintedRailLines paints the frame the real UI paints and returns the body
// lines it put each named session on. Going through View() is the point: a
// test that worked the rail's geometry out for itself would stay green the
// day viewListFrame started passing entryLines something else, which is the
// drift clickRow exists to rule out (#110).
func paintedRailLines(t *testing.T, m *Model, name string) []int {
	t.Helper()
	preparedView(m)
	var lines []int
	for i, row := range railHitRows(m) {
		if row >= 0 && !railRows(m)[row].isGroup && railRows(m)[row].sess.Name == name {
			lines = append(lines, i)
		}
	}
	if len(lines) == 0 {
		t.Fatalf("test setup: %q painted no rail line", name)
	}
	return lines
}

func paintedGroupLines(t *testing.T, m *Model, path string) []int {
	t.Helper()
	preparedView(m)
	var lines []int
	for i, row := range railHitRows(m) {
		if row >= 0 && railRows(m)[row].isGroup && railRows(m)[row].group == path {
			lines = append(lines, i)
		}
	}
	if len(lines) == 0 {
		t.Fatalf("test setup: group %q painted no rail line", path)
	}
	return lines
}

// A click on a session row selects it, reading the geometry the frame
// recorded while painting rather than re-deriving it (#110).
func TestClickSelectsRow(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "beta")

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	updated, _ := m.handleMouse(tea.MouseMsg{
		X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	sess, ok := m.selected()
	if !ok || sess.Name != "alpha" {
		t.Fatalf("click should select alpha, got %q ok=%v", sess.Name, ok)
	}
	if !m.onList() {
		t.Fatalf("first click should select, not focus, screen = %v", m.screenID())
	}
}

// Double click attaches. The pointer names the row, not the cursor: a
// wheel notch, a j or the poll can walk the cursor away between the two
// presses.
func TestDoubleClickAttachesTheRowUnderThePointer(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	press := tea.MouseMsg{X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	updated, _ := m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	m.selectSessionRow(t, "beta")

	updated, _ = m.handleMouse(press)
	m = updated.(*Model)
	if got := attachQueued(t, m); got != "alpha" {
		t.Fatalf("double click should attach the row it landed on, got %q, err = %q", got, m.errBar.text)
	}
}

// The pair is matched on the row's identity: a rebuild between the presses
// renumbers m.rows, so an index that meant this row can mean another.
func TestDoubleClickPairsAcrossARebuild(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	dir := t.TempDir()
	for _, group := range []string{"aaa", "zzz"} {
		if err := m.services.store.CreateGroup(group, dir); err != nil {
			t.Fatalf("group %s: %v", group, err)
		}
	}
	m.applyCmd(t, m.refreshCmd())
	createSession(t, m, "filler", dir, "aaa")
	createSession(t, m, "target", dir, "zzz")

	y0 := m.listChromeRows()
	line := paintedRailLines(t, m, "target")[0]
	updated, _ := m.handleMouse(tea.MouseMsg{
		X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	before := m.rail.Cursor()

	// Folding the group above target drops every row under it an index.
	m.rail.SetCollapsed("aaa", true)
	m.rebuildRows()
	line = paintedRailLines(t, m, "target")[0]
	m.selectSessionRow(t, "target")
	if m.rail.Cursor() == before {
		t.Fatal("test setup: folding should have renumbered target's row")
	}
	setRailCursor(m, before)

	updated, _ = m.handleMouse(tea.MouseMsg{
		X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	if got := attachQueued(t, m); got != "target" {
		t.Fatalf("the pair should survive a rebuild and attach target, got %q, err = %q", got, m.errBar.text)
	}
}

func TestDoubleClickAttachesTheRowJustSelected(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "beta")

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	press := tea.MouseMsg{X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	updated, _ := m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.onList() {
		t.Fatalf("first click should select, screen = %v", m.screenID())
	}
	updated, _ = m.handleMouse(press)
	m = updated.(*Model)
	if got := attachQueued(t, m); got != "alpha" {
		t.Fatalf("double click should attach alpha, got %q, err = %q", got, m.errBar.text)
	}
}

func TestSlowSecondClickDoesNotFocus(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	m.selectSessionRow(t, "alpha")

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	press := tea.MouseMsg{X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	ctx := m.railMouseContext()
	ctx.Now = time.Now().Add(-time.Second)
	m.applyRailDecision(m.rail.Mouse(press, m.layout.displayedRail, ctx))
	m.drainEffects(t)
	updated, _ := m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.onList() {
		t.Fatalf("a slow second click should not focus, screen = %v", m.screenID())
	}
}

func TestClickOnSelectedGroupTogglesCollapse(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	dir := t.TempDir()
	if err := m.services.store.CreateGroup("work", dir); err != nil {
		t.Fatalf("group: %v", err)
	}
	m.applyCmd(t, m.refreshCmd())
	createSession(t, m, "alpha", dir, "work")
	m.selectGroupRow(t, "work")
	if m.rail.IsCollapsed("work") {
		t.Fatal("test setup: work should start open")
	}

	line := paintedGroupLines(t, m, "work")[0]
	y0 := m.listChromeRows()
	press := tea.MouseMsg{X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	updated, _ := m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	updated, _ = m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.rail.IsCollapsed("work") {
		t.Fatal("double click on the selected group should fold it")
	}

	line = paintedGroupLines(t, m, "work")[0]
	y0 = m.listChromeRows()
	press.Y = y0 + line
	updated, _ = m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	updated, _ = m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if m.rail.IsCollapsed("work") {
		t.Fatal("a second double click should unfold it")
	}
}

// The rail's last painted column carries row text, so it belongs to the
// row under it.
func TestClickOnRailLastColumnSelectsRow(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "beta")

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	updated, _ := m.handleMouse(tea.MouseMsg{
		X: m.layout.width - 1, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	if sess, ok := m.selected(); !ok || sess.Name != "alpha" {
		t.Fatalf("click on the rail's last column should select alpha, got %q ok=%v", sess.Name, ok)
	}
}

// A comfortable entry paints two or three lines; a click on any of them
// should select the entry, not whatever railHits index that physical line
// would be under a compact row.
func TestClickSelectsRowAcrossComfortableLines(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.prefs.comfortableRows = true
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "alpha")

	alphaLines := paintedRailLines(t, m, "alpha")
	if len(alphaLines) < 2 {
		t.Fatalf("test setup: comfortable alpha should paint 2+ lines, got %d", len(alphaLines))
	}
	m.selectSessionRow(t, "beta")
	alphaLines = paintedRailLines(t, m, "alpha")

	y0 := m.listChromeRows()
	last := alphaLines[len(alphaLines)-1]
	updated, _ := m.handleMouse(tea.MouseMsg{
		X: 2, Y: y0 + last, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	if sess, ok := m.selected(); !ok || sess.Name != "alpha" {
		t.Fatalf("clicking alpha's later line should still select alpha, got %q ok=%v", sess.Name, ok)
	}
}

// A "N more" counter answers for the entry it is hiding: clicking it steps
// the window onto that entry rather than handing back the same frame. The
// rail's other chrome — the search field, the filter badges, the padding
// and the meters — still picks nothing.
func TestClickOnMoreCounterSelectsTheRowItHides(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	for _, name := range []string{"one", "two", "three", "four", "five", "six", "seven"} {
		createSession(t, m, name, t.TempDir(), "")
	}
	m.selectSessionRow(t, "one")
	// A rail too short for seven sessions, so the window has to keep a
	// counter for the ones it is leaving out.
	m.layout.height = 12
	frame := uitheme.SplitLines(preparedView(m))

	y0 := m.listChromeRows()
	counter, target := -1, -1
	for i, row := range railHitRows(m) {
		if row >= 0 && strings.Contains(frame[y0+i], "more") {
			counter, target = i, row
		}
	}
	if counter < 0 {
		t.Fatal("test setup: a short rail should paint a counter")
	}
	for i, row := range railHitRows(m) {
		if i != counter && row == target {
			t.Fatalf("test setup: row %d is painted at line %d, so it is not hidden", target, i)
		}
	}

	updated, _ := m.handleMouse(tea.MouseMsg{
		X: 2, Y: y0 + counter, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	if m.rail.Cursor() != target {
		t.Fatalf("clicking the counter should select row %d, cursor = %d", target, m.rail.Cursor())
	}
}

// A counter only scrolls: its row is off screen, so a click there neither
// focuses that row on release nor finds a handle or a menu button on it.
func TestMoreCounterOnlyScrolls(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	for _, name := range []string{"one", "two", "three", "four", "five", "six", "seven"} {
		createSession(t, m, name, t.TempDir(), "")
	}
	m.layout.height = 12
	preparedView(m)
	if m.layout.displayedRail.Width == 0 {
		t.Fatal("test setup: the frame should record the rail width")
	}
	for _, x := range []int{2, 4, m.layout.displayedRail.Width} {
		m.selectSessionRow(t, "one")
		frame := uitheme.SplitLines(preparedView(m))
		y0 := m.listChromeRows()
		counter := -1
		for i, row := range railHitRows(m) {
			if row >= 0 && strings.Contains(frame[y0+i], "more") {
				counter = i
			}
		}
		if counter < 0 {
			t.Fatal("test setup: a short rail should paint a counter")
		}
		at := tea.MouseMsg{X: x, Y: y0 + counter, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
		updated, _ := m.handleMouse(at)
		m.drainEffects(t)
		m = updated.(*Model)
		m.drainEffects(t)
		at.Action = tea.MouseActionRelease
		updated, _ = m.handleMouse(at)
		m.drainEffects(t)
		m = updated.(*Model)
		m.drainEffects(t)
		if !m.onList() || m.rail.MenuOpen() || m.rail.Reordering() {
			t.Fatalf("a click on the counter at x=%d should only scroll, screen = %v menu = %v lifted = %v", x, m.screenID(), m.rail.MenuOpen(), m.rail.Reordering())
		}
	}
}

// Chrome the rail paints for its own sake carries no row, so a click there
// picks nothing rather than misattributing to whichever row happens to sit
// at that index.
func TestClickOnRailChromeDoesNotSelect(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "beta")
	m.rail.SetSearch(m.rail.Search(), true)
	before := m.rail.Cursor()
	preparedView(m)

	line := -1
	for i, row := range railHitRows(m) {
		if row < 0 {
			line = i
			break
		}
	}
	if line < 0 {
		t.Fatal("test setup: the search field should paint chrome lines")
	}
	y0 := m.listChromeRows()
	updated, cmd := m.handleMouse(tea.MouseMsg{
		X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	if m.rail.Cursor() != before || cmd != nil {
		t.Fatal("clicking a chrome line should not move the cursor")
	}
}

// Wheel events must be consumed by the app so the host terminal cannot
// scroll the TUI away, and in the list moves the session cursor the same
// way an arrow key would (#110).
func TestWheelMovesListCursor(t *testing.T) {
	m := &Model{
		layout: layoutState{
			width:  80,
			height: 24,
		},
		services:  services{listKeys: keybind.DefaultList()},
		workspace: workspace{sessions: []store.Session{{ID: "fixture-b", Name: "fixture"}}},
		rail:      railModelCount(2, 0, false),
	}
	updated, _ := m.handleMouse(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	if m.rail.Cursor() != 1 {
		t.Fatalf("wheel down: cursor = %d want 1", m.rail.Cursor())
	}
	updated, _ = m.handleMouse(tea.MouseMsg{
		Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress,
	})
	m = updated.(*Model)
	m.drainEffects(t)
	if m.rail.Cursor() != 0 {
		t.Fatalf("wheel up: cursor = %d want 0", m.rail.Cursor())
	}
}

// The wheel keeps moving the selection while the search field or the quick
// prompt is open. Search otherwise narrows the list to rows none of the
// pointer or the keyboard can reach, and the quick bar retargets on up/down
// the way its own footer advertises.
func TestWheelMovesCursorWhileSearchingOrPrompting(t *testing.T) {
	for name, m := range map[string]*Model{
		"searching": {layout: layoutState{width: 80, height: 24}, rail: railModelCount(2, 0, true)},
		"quick bar": {layout: layoutState{width: 80, height: 24}, rail: railModelCount(2, 0, false)},
	} {
		t.Run(name, func(t *testing.T) {
			if name == "quick bar" {
				pushQuick(m)
			}
			updated, _ := m.handleMouse(tea.MouseMsg{
				Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
			})
			m := updated.(*Model)
			if m.rail.Cursor() != 1 {
				t.Fatalf("wheel should have moved the cursor to 1, got %d", m.rail.Cursor())
			}
		})
	}
}

// Search keeps click-to-select for the same reason it keeps the wheel.
func TestClickSelectsRowWhileSearching(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "beta")
	m.rail.SetSearch(m.rail.Search(), true)

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	at := tea.MouseMsg{X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	updated, _ := m.handleMouse(at)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	at.Action = tea.MouseActionRelease
	updated, _ = m.handleMouse(at)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if sess, ok := m.selected(); !m.onList() || !ok || sess.Name != "alpha" {
		t.Fatalf("click should select alpha and stay in the list, screen = %v got %q", m.screenID(), sess.Name)
	}
}

// A press search swallows opens no run, so the first press after it closes
// is a plain select rather than the second half of a full layout pair.
func TestClickWhileSearchingOpensNoClickRun(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	m.selectSessionRow(t, "alpha")
	m.rail.SetSearch(m.rail.Search(), true)

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	press := tea.MouseMsg{X: 2, Y: y0 + line, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
	updated, _ := m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)

	m.rail.SetSearch(m.rail.Search(), false)
	updated, _ = m.handleMouse(press)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.onList() {
		t.Fatalf("one press after search closes must not focus, screen = %v", m.screenID())
	}
}

// Search owns Enter, so a double click does not attach while it is open:
// it must not steal the key the field is waiting for.
func TestClicksDoNotAttachWhileSearching(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	m.selectSessionRow(t, "alpha")
	m.rail.SetSearch(m.rail.Search(), true)

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	at := tea.MouseMsg{X: 2, Y: y0 + line, Button: tea.MouseButtonLeft}
	for i := 0; i < 2; i++ {
		for _, action := range []tea.MouseAction{tea.MouseActionPress, tea.MouseActionRelease} {
			at.Action = action
			updated, _ := m.handleMouse(at)
			m = updated.(*Model)
		}
	}
	if !m.onList() || attachQueued(t, m) != "" {
		t.Fatalf("clicks must not attach while searching, screen = %v", m.screenID())
	}
}

// One flick of the wheel is several notches, so the wheel clamps where the
// keyboard wraps: going off the end must not fling the selection to the
// far one and aim every key after it somewhere the user never looked.
func TestWheelClampsAtBothEnds(t *testing.T) {
	m := &Model{layout: layoutState{width: 80, height: 24}, services: services{listKeys: keybind.DefaultList()}, rail: railModelCount(3, 2, false)}
	for i := 0; i < 3; i++ {
		updated, _ := m.handleMouse(tea.MouseMsg{
			Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress,
		})
		m = updated.(*Model)
		m.drainEffects(t)
	}
	if m.rail.Cursor() != 2 {
		t.Fatalf("wheel down off the bottom: cursor = %d want it held at 2", m.rail.Cursor())
	}
	setRailCursor(m, 0)
	for i := 0; i < 3; i++ {
		updated, _ := m.handleMouse(tea.MouseMsg{
			Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress,
		})
		m = updated.(*Model)
		m.drainEffects(t)
	}
	if m.rail.Cursor() != 0 {
		t.Fatalf("wheel up off the top: cursor = %d want it held at 0", m.rail.Cursor())
	}
	// The keyboard still wraps: clamping is the wheel's alone.
	m.moveCursor(-1)
	if m.rail.Cursor() != 2 {
		t.Fatalf("arrow up should still wrap to 2, cursor = %d", m.rail.Cursor())
	}
}

// messagesCell finds the column and screen row the word "unread" was
// painted on in the last frame: the card's legend in the split, the badge
// in full screen.
func messagesCell(t *testing.T, m *Model) (x, y int) {
	t.Helper()
	if len(m.activeNotices()) == 0 {
		withNotice(m, "refused")
	}
	for row, line := range strings.Split(ansi.Strip(preparedView(m)), "\n") {
		if col := strings.Index(line, "unread"); col >= 0 {
			return ansi.StringWidth(line[:col]), row
		}
	}
	for _, hit := range m.layout.keyHits {
		if m.services.listKeys.Binding(keybind.Messages).Has(hit.Key.String()) {
			return hit.X0, hit.Y
		}
	}
	t.Fatal("test setup: the frame painted no messages card or key")
	return 0, 0
}

func leftPress(m *Model, x, y int) *Model {
	updated, _ := m.handleMouse(tea.MouseMsg{
		X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft,
	})
	return updated.(*Model)
}

func leftRelease(m *Model, x, y int) *Model {
	m, _ = leftReleaseCmd(m, x, y)
	return m
}

func leftReleaseCmd(m *Model, x, y int) (*Model, tea.Cmd) {
	updated, cmd := m.handleMouse(tea.MouseMsg{
		X: x, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft,
	})
	return updated.(*Model), cmd
}

// leftClick presses and releases the left button over x, y.
func leftClick(m *Model, x, y int) *Model {
	return leftRelease(leftPress(m, x, y), x, y)
}

func leftClickCmd(m *Model, x, y int) (*Model, tea.Cmd) {
	return leftReleaseCmd(leftPress(m, x, y), x, y)
}

// wheel turns the wheel one notch over x, y.
func wheel(m *Model, button tea.MouseButton, x, y int) *Model {
	updated, _ := m.handleMouse(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: button})
	return updated.(*Model)
}

// wheelOnCard turns the wheel one notch over the middle of the open card.
func wheelOnCard(m *Model, button tea.MouseButton) *Model {
	preparedView(m)
	card := m.layout.card
	return wheel(m, button, (card.left+card.right)/2, (card.top+card.bottom)/2)
}

// paintedAt is where text starts on the frame row that paints it, the
// lowest such row when fromBottom.
func paintedAt(m *Model, text string, fromBottom bool) (x, y int, ok bool) {
	rows := strings.Split(ansi.Strip(preparedView(m)), "\n")
	for i := range rows {
		row := i
		if fromBottom {
			row = len(rows) - 1 - i
		}
		if col := strings.Index(rows[row], text); col >= 0 {
			return ansi.StringWidth(rows[row][:col]), row, true
		}
	}
	return 0, 0, false
}

// paintedCell is where label starts on the lowest frame row that paints
// it, which is a footer or a card's foot before any body row.
func paintedCell(t *testing.T, m *Model, label string) (x, y int) {
	t.Helper()
	x, y, ok := paintedAt(m, label, true)
	if !ok {
		t.Fatalf("test setup: the frame painted no %q:\n%s", label, ansi.Strip(m.View()))
	}
	return x, y
}

// bodyCell is where text starts on the highest frame row that paints it,
// which is a card's body before its foot.
func bodyCell(t *testing.T, m *Model, text string) (x, y int) {
	t.Helper()
	x, y, ok := paintedAt(m, text, false)
	if !ok {
		t.Fatalf("test setup: the frame painted no %q:\n%s", text, ansi.Strip(m.View()))
	}
	return x, y
}

func paints(m *Model, text string) bool {
	_, _, ok := paintedAt(m, text, false)
	return ok
}

// clickLabel clicks a key a footer or a card's foot painted.
func clickLabel(t *testing.T, m *Model, label string) *Model {
	t.Helper()
	x, y := paintedCell(t, m, label)
	m = leftClick(m, x, y)
	m.drainEffects(t)
	return m
}

// clickBody clicks where a card's body painted text.
func clickBody(t *testing.T, m *Model, text string) (*Model, tea.Cmd) {
	t.Helper()
	x, y := bodyCell(t, m, text)
	return leftClickCmd(m, x, y)
}

func TestClickOnMessagesCardOpensNotices(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	x, y := messagesCell(t, m)
	if m = leftClick(m, x, y); !m.noticesOpen() {
		t.Fatalf("click on the card should open messages, screen = %v", m.screenID())
	}
}

func TestMessagesLegendOpensNoticesByMouseAndKey(t *testing.T) {
	t.Parallel()
	for _, width := range []int{30, 60, 120} {
		for _, binding := range []string{"M", "alt+m"} {
			m := buildModel(t)
			m.layout.width, m.layout.height = width, 34
			m.prefs.hideStats = true
			m.services.listKeys = m.services.listKeys.With(keybind.Messages, bindingOf(t, binding))
			x, y := messagesCell(t, m)
			if _, ok := m.layout.keyHits.At(x, y); !ok {
				t.Fatalf("width=%d binding=%s: painted messages entry misses its hit box %+v", width, binding, m.layout.keyHits)
			}
			if m = leftClick(m, x, y); !m.noticesOpen() {
				t.Fatalf("click on messages should open notices, screen = %v", m.screenID())
			}
			m.handleKey(key("esc"))
			msg := key(binding)
			if binding == "alt+m" {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}, Alt: true}
			}
			// A footer too narrow for words paints the entry as its key
			// alone, so the entry is found by the hit that presses it and
			// read back from the cells under that hit.
			rows := strings.Split(ansi.Strip(preparedView(m)), "\n")
			found := false
			var entry keyHit
			for _, hit := range m.layout.keyHits {
				if hit.Key.String() == msg.String() && (!found || hit.Y > entry.Y) {
					entry, found = hit, true
				}
			}
			if !found {
				t.Fatalf("width=%d binding=%s: no hit opens messages %+v", width, binding, m.layout.keyHits)
			}
			if painted := strings.TrimSpace(ansi.Cut(rows[entry.Y], entry.X0, entry.X1)); !strings.HasPrefix(painted, m.listGlyph(keybind.Messages)) {
				t.Fatalf("width=%d binding=%s: the messages hit covers %q, not its key %q", width, binding, painted, m.listGlyph(keybind.Messages))
			}
			x, y = entry.X0, entry.Y
			if m = leftClick(m, x, y); !m.noticesOpen() {
				t.Fatalf("click on messages should open notices, screen=%v", m.screenID())
			}
			m.handleKey(key("esc"))
			m.handleKey(msg)
			if !m.noticesOpen() {
				t.Fatalf("%s should open notices, screen = %v", binding, m.screenID())
			}
			m.handleKey(key("esc"))
			for _, n := range m.activeNotices() {
				m.dismissNotice(n.ID)
			}
			preparedView(m)
			if footer := ansi.Strip(m.viewFooter()); strings.Contains(footer, "unread") {
				t.Fatalf("with nothing unread the legend must give its room back:\n%s", footer)
			}
			if m = leftClick(m, x, y); m.noticesOpen() {
				t.Fatal("a click where the entry was must not open Messages")
			}
			m.handleKey(key("esc"))
			m.handleKey(msg)
			if !m.noticesOpen() {
				t.Fatalf("%s should still open an empty Messages, screen = %v", binding, m.screenID())
			}
		}
	}
}

func TestMessagesLegendClickOnlyOpensFromTheList(t *testing.T) {
	for _, change := range []struct {
		name  string
		apply func(*Model)
		ready func(*Model) bool
	}{
		{"search", func(m *Model) { m.rail.SetSearch(m.rail.Search(), true) }, func(m *Model) bool { return m.rail.Searching() }},
		{"quick prompt", func(m *Model) { m.openQuickMode() }, func(m *Model) bool { return (m.quick() != nil) }},
		{"messages key off", func(m *Model) { m.services.listKeys = m.services.listKeys.With(keybind.Messages, bindingOf(t)) }, func(*Model) bool { return true }},
		{"settings", func(m *Model) { m.openSettings() }, func(m *Model) bool { return m.settingsOpen() }},
		{"short terminal", func(m *Model) { m.layout.height = 2 }, func(*Model) bool { return true }},
	} {
		t.Run(change.name, func(t *testing.T) {
			m := buildModel(t)
			m.layout.width, m.layout.height = 120, 34
			x, y := messagesCell(t, m)
			change.apply(m)
			if !change.ready(m) {
				t.Fatalf("test setup: %s did not take, screen = %v err = %q", change.name, m.screenID(), m.errBar.text)
			}
			preparedView(m)
			if m = leftClick(m, x, y); m.noticesOpen() {
				t.Fatal("a click where messages was painted opened notices")
			}
		})
	}
}

func TestClickOnMetersBesideTheCardDoesNothing(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	_, y := messagesCell(t, m)
	if m = leftClick(m, 2, y); !m.onList() {
		t.Fatalf("click on the meters should stay on the list, screen = %v", m.screenID())
	}
}

func TestClickOnFullScreenBadgeOpensNotices(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	x, y := messagesCell(t, m)
	if m = leftClick(m, x, y); !m.noticesOpen() {
		t.Fatalf("click on the badge should open messages, screen = %v", m.screenID())
	}
	m.base = baseList
	m.clearSurfaces()
	if m = leftClick(m, 2, y); !m.onList() {
		t.Fatalf("click on the readings should stay on the list, screen = %v", m.screenID())
	}
}

func TestClickOnMessagesCardWhileSearchingKeepsTheField(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	m.rail.SetSearch(m.rail.Search(), true)
	x, y := messagesCell(t, m)
	if m = leftClick(m, x, y); !m.onList() || !m.rail.Searching() {
		t.Fatalf("search owns the click, screen = %v searching = %v", m.screenID(), m.rail.Searching())
	}
}

func TestSingleClickReleasedOnAnotherRowDoesNotFocus(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")

	y0 := m.listChromeRows()
	alpha := paintedRailLines(t, m, "alpha")[0]
	beta := paintedRailLines(t, m, "beta")[0]
	updated, _ := m.handleMouse(tea.MouseMsg{X: 2, Y: y0 + alpha, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	updated, _ = m.handleMouse(tea.MouseMsg{X: 2, Y: y0 + beta, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	updated, _ = m.handleMouse(tea.MouseMsg{X: 2, Y: y0 + alpha, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.onList() {
		t.Fatalf("a press dragged off its row should not focus, screen = %v", m.screenID())
	}
}

func TestSingleClickInFullLayoutOnlySelects(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")

	line := paintedRailLines(t, m, "alpha")[0]
	y0 := m.listChromeRows()
	at := tea.MouseMsg{X: 2, Y: y0 + line, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	updated, _ := m.handleMouse(at)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	at.Action = tea.MouseActionRelease
	updated, _ = m.handleMouse(at)
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.onList() {
		t.Fatalf("full layout keeps click as select, screen = %v", m.screenID())
	}
}

func TestClicksWorkWhileTheQuickBarIsOpen(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.openQuickMode()
	if m.quick() == nil {
		t.Fatal("test setup: quick bar should open")
	}
	y0 := m.listChromeRows()
	beta := y0 + paintedRailLines(t, m, "beta")[0]

	updated, _ := m.handleMouse(tea.MouseMsg{X: m.layout.displayedRail.Width, Y: beta, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m.drainEffects(t)
	m = updated.(*Model)
	m.drainEffects(t)
	if !m.rail.MenuOpen() || (m.quick() == nil) {
		t.Fatalf("%s should open the menu over the open quick bar, menu = %v quick = %v", rowMenuGlyph, m.rail.MenuOpen(), (m.quick() != nil))
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m.drainEffects(t)
	if m.rail.MenuOpen() || (m.quick() == nil) {
		t.Fatalf("esc should close the menu first and leave the bar, menu = %v quick = %v", m.rail.MenuOpen(), (m.quick() != nil))
	}

	m = liftByHandle(t, m, "beta")
	if m.viewFooter() == "" || !strings.Contains(ansi.Strip(m.viewFooter()), "Reorder") {
		t.Fatal("a lifted row should own the footer over the quick bar")
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m.drainEffects(t)

	at := tea.MouseMsg{X: 8, Y: y0 + paintedRailLines(t, m, "alpha")[0], Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	updated, _ = m.handleMouse(at)
	m = updated.(*Model)
	m.drainEffects(t)
	at.Action = tea.MouseActionRelease
	updated, _ = m.handleMouse(at)
	m = updated.(*Model)
	m.drainEffects(t)
	if sess, ok := m.selected(); !ok || sess.Name != "alpha" {
		t.Fatalf("a row click should select alpha over the bar, got %q ok=%v", sess.Name, ok)
	}
}

// A corner of the terminal is backdrop under every centered card.
func clickBackdrop(t *testing.T, m *Model) *Model {
	t.Helper()
	preparedView(m)
	if m.layout.card.holds(0, 0) {
		t.Fatalf("test setup: the card covers the corner: %+v", m.layout.card)
	}
	m = leftClick(m, 0, 0)
	m.drainEffects(t)
	return m
}

func TestBackdropClickClosesACardTheWayEscDoes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		open func(*Model)
	}{
		{"help", func(m *Model) { m.openHelp() }},
		{"messages", func(m *Model) { m.openNotices("") }},
		{"settings", func(m *Model) { m.applyCmd(t, m.openSettings()) }},
		{"confirm", func(m *Model) {
			m.openConfirm(confirmTarget{prompt: uidialogs.Confirm{Action: uidialogs.ConfirmKill, Label: "kill alpha? frees its RAM."}, lifecycle: uilifecycle.Target{Action: uilifecycle.Action(uidialogs.ConfirmKill)}})
		}},
		{"move", func(m *Model) { m.openMove() }},
	} {
		m := buildModel(t)
		m.layout.width, m.layout.height = 120, 34
		createSession(t, m, "alpha", t.TempDir(), "")
		m.selectSessionRow(t, "alpha")
		tc.open(m)
		if m.listClear() {
			t.Fatalf("%s: test setup did not open the card, err %q", tc.name, m.errBar.text)
		}
		if m = clickBackdrop(t, m); !m.listClear() {
			t.Errorf("%s: a backdrop click should close the card, screen = %v", tc.name, m.screenID())
		}
		if confirmOpen(m) {
			t.Errorf("%s: a backdrop click on a confirm is its cancel, target = %+v", tc.name, pendingConfirm(m))
		}
	}
}

func TestBackdropClickKeepsATypedDraft(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		open func(*Model)
	}{
		{"new session", func(m *Model) { m.applyCmd(t, m.openForm()) }},
		{"new group", func(m *Model) { m.openGroupForm() }},
		{"fork", func(m *Model) {
			m.push(uidialogs.NewForkSurface(forkRoot{m, store.Session{Name: "alpha"}}, "alpha", "", 1))
		}},
	} {
		m := buildModel(t)
		m.layout.width, m.layout.height = 120, 34
		createSessionOn(t, m, "alpha", "claude", t.TempDir())
		m.selectSessionRow(t, "alpha")
		tc.open(m)
		if len(m.surfaces) != 1 {
			t.Fatalf("%s: test setup put %d surfaces up, err %q", tc.name, len(m.surfaces), m.errBar.text)
		}
		if m = clickBackdrop(t, m); len(m.surfaces) != 1 {
			t.Errorf("%s: a stray click closed the draft", tc.name)
		}
	}
}

func TestBackdropClickKeepsTheSettingsEditorField(t *testing.T) {
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	editorSettings(t, m, "code")
	m.pressInSettings(t, tea.KeyMsg{Type: tea.KeyLeft})
	m.pressInSettings(t, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.settingsCard().State().Editor.Typing() {
		t.Fatal("test setup: enter on custom opens the field")
	}
	if m = clickBackdrop(t, m); !m.settingsOpen() || !m.settingsCard().State().Editor.Typing() {
		t.Fatalf("a stray click should keep the typed line open, screen = %v typing = %v", m.screenID(), m.settingsCard().State().Editor.Typing())
	}
}

// A click inside the card that lands on nothing it paints leaves it open.
func TestClickOnACardsOwnBlankStaysOpen(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	m.openHelp()
	preparedView(m)
	if m = leftClick(m, m.layout.card.left+1, m.layout.card.top+1); !helpOpen(m) {
		t.Fatalf("a click on the card's own edge should leave it open, screen = %v", m.screenID())
	}
}

func TestBackdropClickKeepsASavedPromptDraft(t *testing.T) {
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	m.applyCmd(t, m.openSettings())
	m.applyCmd(t, m.runSettingsResult(m.settingsCard().State(), m.settingsCard().State().OpenPrompts()))
	m.pressInSettings(t, runeKey("n"))
	if !m.settingsCard().State().Prompts.Editing() {
		t.Fatal("test setup: n opens the prompt editor")
	}
	if m = clickBackdrop(t, m); !m.settingsOpen() || !m.settingsCard().State().Prompts.Editing() {
		t.Fatalf("a stray click should keep the prompt being typed, screen = %v", m.screenID())
	}
}

// A rename typed into its row holds the list still under the mouse.
func TestRenameHoldsTheListUnderTheMouse(t *testing.T) {
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	createSession(t, m, "alpha", t.TempDir(), "")
	createSession(t, m, "beta", t.TempDir(), "")
	m.selectSessionRow(t, "alpha")
	m.openRename()
	x, y := bodyCell(t, m, "beta")
	m = leftClick(m, x, y)
	m = wheel(m, tea.MouseButtonWheelDown, x, y)
	if row, ok := m.selectedRow(); !renameOpen(m) || !ok || row.sess.Name != "alpha" {
		t.Fatalf("the mouse moved the list under a rename: mode %v row %+v", m.screenID(), row.sess.Name)
	}
}

// A card answers a click when the button comes back up over the spot the
// press landed on, so a press on the backdrop released over the card
// closes nothing.
func TestACardClickRunsOnTheRelease(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	m.openHelp()
	preparedView(m)
	card := m.layout.card
	if m = leftPress(m, 0, 0); !helpOpen(m) {
		t.Fatalf("the press alone closed the card, screen = %v", m.screenID())
	}
	if m = leftRelease(m, card.left+1, card.top+1); !helpOpen(m) {
		t.Fatalf("a release over the card closed it, screen = %v", m.screenID())
	}
	if m = leftClick(m, 0, 0); !m.onList() {
		t.Fatalf("a click on the backdrop should close the card, screen = %v", m.screenID())
	}
}

// mouseOpeners mounts every surface the way the user does, on a list with
// a selected session. A surface type missing here fails, so a new surface
// is driven by the mouse on purpose.
var mouseOpeners = map[string]func(t *testing.T, m *Model){
	"dialogs.ConfirmSurface": func(t *testing.T, m *Model) {
		m.openConfirm(confirmTarget{prompt: uidialogs.Confirm{Action: uidialogs.ConfirmKill, Label: "kill alpha?"}, lifecycle: uilifecycle.Target{Action: uilifecycle.Action(uidialogs.ConfirmKill)}})
	},
	"dialogs.MoveSurface":   func(t *testing.T, m *Model) { m.openMove() },
	"dialogs.RenameSurface": func(t *testing.T, m *Model) { m.openRename() },
	"dialogs.ForkSurface": func(t *testing.T, m *Model) {
		source := railSelectedSession(m)
		m.push(uidialogs.NewForkSurface(forkRoot{m, source}, source.Name, source.Group, 1))
	},
	"dialogs.LaunchHintSurface": func(t *testing.T, m *Model) {
		m.openLaunchHint(uiinstall.Fix{Text: "tool is missing", Command: "true", Binary: "tool"})
	},
	"help.Surface": func(t *testing.T, m *Model) { m.openHelp() },
	"notices.Surface": func(t *testing.T, m *Model) {
		withNotice(m, "refused")
		m.Update(key("M"))
	},
	"onboarding.Surface":       func(t *testing.T, m *Model) { m.openSetup(true) },
	"settings.Surface":         func(t *testing.T, m *Model) { m.applyCmd(t, m.openSettings()) },
	"dialogs.GroupFormSurface": func(t *testing.T, m *Model) { m.openGroupForm() },
	"connections.Surface":      func(t *testing.T, m *Model) { m.openConnectionDialog(remote.Connection{}) },
	"quick.Surface":            func(t *testing.T, m *Model) { m.applyCmd(t, m.openQuickMode()) },
	"spawn.Surface":            func(t *testing.T, m *Model) { m.applyCmd(t, m.openForm()) },
	"review.Surface": func(t *testing.T, m *Model) {
		m.base = baseReview
		m.push(uireview.NewSurface(reviewRoot{m}, []uireview.PickerRow{{Label: "main"}}, "branch", uireview.PickRepo, "", uireview.PickerSource{}, "", uireview.RenderContext{}, ""))
	},
}

// Every surface answers the mouse: a wheel and a click over it neither
// panic nor touch the stack under it.
func TestEverySurfaceAnswersTheMouse(t *testing.T) {
	for _, name := range declaredSurfaces(t) {
		open, ok := mouseOpeners[name]
		if !ok {
			t.Errorf("%s has no opener in mouseOpeners: drive it by the mouse here", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			m := buildModel(t)
			m.layout.width, m.layout.height = 120, 40
			createSession(t, m, "alpha", t.TempDir(), "")
			m.selectSessionRow(t, "alpha")
			open(t, m)
			top, ok := m.top()
			if !ok || surfaceName(top) != name {
				t.Fatalf("the opener did not put %s on top, stack %d, err %q", name, len(m.surfaces), m.errBar.text)
			}
			below := len(m.surfaces) - 1
			preparedView(m)
			x, y := m.layout.width/2, m.layout.height/2
			for _, msg := range []tea.MouseMsg{
				{X: x, Y: y, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress},
				{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
				{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
			} {
				m.Update(msg)
				if len(m.surfaces) < below {
					t.Fatalf("a mouse event over %s popped the stack under it", name)
				}
			}
		})
	}
	for name := range mouseOpeners {
		if !contains(declaredSurfaces(t), name) {
			t.Errorf("mouseOpeners names %s, which no longer implements surface", name)
		}
	}
}

// surfaceName names a surface the way declaredSurfaces does: bare in the
// root, qualified by its package in a feature's.
func surfaceName(s uisurface.Surface) string {
	typ := reflect.TypeOf(s).Elem()
	if pkg := path.Base(typ.PkgPath()); pkg != "ui" {
		return pkg + "." + typ.Name()
	}
	return typ.Name()
}

// declaredSurfaces lists the types that declare a Leave(uisurface.Host) method on a
// pointer receiver, in the root and in the feature packages under it: every
// surface does, and nothing else.
func declaredSurfaces(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	dirs := []string{"."}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "testdata" {
			dirs = append(dirs, entry.Name())
		}
	}
	var names []string
	for _, dir := range dirs {
		for _, file := range parseGoPackage(t, dir, 0) {
			pkg := file.Name.Name
			{
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv == nil || fn.Name.Name != "Leave" || !takesHost(fn) {
						continue
					}
					if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
						if ident, ok := star.X.(*ast.Ident); ok {
							name := ident.Name
							if pkg != "ui" {
								name = pkg + "." + name
							}
							names = append(names, name)
						}
					}
				}
			}
		}
	}
	if len(names) == 0 {
		t.Fatal("found no surfaces")
	}
	return names
}

// takesHost is whether fn's one parameter is the stack's host, which tells a
// surface's Leave from another type's.
func takesHost(fn *ast.FuncDecl) bool {
	params := fn.Type.Params.List
	if len(params) != 1 {
		return false
	}
	if typ, ok := params[0].Type.(*ast.SelectorExpr); ok {
		return typ.Sel.Name == "Host"
	}
	return false
}

func TestSettingsWheelReachesSymbolsAndDocsInShortWindows(t *testing.T) {
	// One row shorter than the symbols line. Save lives in the footer, so the
	// body spends that row on a field.
	for _, size := range [][2]int{{34, 11}, {80, 10}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			m := buildModel(t)
			m.layout.width, m.layout.height = size[0], size[1]
			m.runBatch(t, m.openSettings())
			if paints(m, "symbols") {
				t.Fatal("symbols should start below this viewport")
			}
			for range uisettings.FieldGlyphs {
				m = wheelOnCard(m, tea.MouseButtonWheelDown)
			}
			if m.settingsCard().State().Field != uisettings.FieldGlyphs || !paints(m, "symbols") {
				t.Fatal("wheel did not reveal symbols")
			}
			if m, _ = clickBody(t, m, "symbols"); m.prefs.glyphMode != presentation.GlyphModeBasic {
				t.Fatal("revealed symbols row was not clickable")
			}
			// The environments row between them waits for a profile.
			for range uisettings.FieldDocs - uisettings.FieldGlyphs - 1 {
				m = wheelOnCard(m, tea.MouseButtonWheelDown)
			}
			if m.settingsCard().State().Field != uisettings.FieldDocs || !paints(m, "docs") {
				t.Fatal("wheel did not reveal docs")
			}
			var opened string
			openBrowser = func(url string) error { opened = url; return nil }
			t.Cleanup(func() { openBrowser = uiterminal.DefaultOpenBrowser })
			m, docs := clickBody(t, m, "docs")
			m.runBatch(t, docs)
			if opened != docsURL {
				t.Fatal("docs click did not follow the scrolled row")
			}
			// The environments row between them waits for a profile.
			for range uisettings.FieldDocs - uisettings.FieldGlyphs - 1 {
				m = wheelOnCard(m, tea.MouseButtonWheelUp)
			}
			if m.settingsCard().State().Field != uisettings.FieldGlyphs || !paints(m, "symbols") {
				t.Fatal("wheel up did not return to symbols")
			}
		})
	}
}

func TestSettingsWheelLeavesSubpanelsAlone(t *testing.T) {
	for _, panel := range []string{"CLIs", "keys", "editor"} {
		t.Run(panel, func(t *testing.T) {
			m := buildModel(t)
			m.runBatch(t, m.openSettings())
			switch panel {
			case "CLIs":
				m.settingsCard().State().Field = uisettings.FieldCLIs
				m.handleSettingsKey(key("enter"))
			case "keys":
				m.settingsCard().State().Field = uisettings.FieldKeybindings
				m.handleSettingsKey(key("enter"))
			case "editor":
				m.settingsCard().State().Field = uisettings.FieldEditor
				m.settingsCard().State().Editor = uisettings.NewEditorRow("custom --test", nil)
				m.handleSettingsKey(key("enter"))
			}
			field := m.settingsCard().State().Field
			for _, button := range []tea.MouseButton{tea.MouseButtonWheelDown, tea.MouseButtonWheelUp} {
				if m = wheelOnCard(m, button); m.settingsCard().State().Field != field {
					t.Fatal("wheel in a subpanel moved the parent Settings field")
				}
			}
		})
	}
}

func TestClickOnListFooterGroupKeyOpensTheCard(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 320, 40
	createSession(t, m, "alpha", t.TempDir(), "")
	if m = clickLabel(t, m, "g group"); !m.groupFormOpen() {
		t.Fatalf("click on the group key did not open the card (err %q)", m.errBar.text)
	}
}

func TestClickOnListFooterSettingsKeyOpensTheCard(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 320, 40
	createSession(t, m, "alpha", t.TempDir(), "")
	if m = clickLabel(t, m, "s settings"); !m.settingsOpen() {
		t.Fatalf("click on the settings key did not open the card (err %q)", m.errBar.text)
	}
}

func TestClickOnListFooterKeysOpensTheirModals(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		label string
		open  func(*Model) bool
	}{
		{"n new", formOpen},
		{"g group", func(m *Model) bool { return m.groupFormOpen() }},
		{"? keys", func(m *Model) bool { _, ok := surfaceOf[*uihelp.Surface](m); return ok }},
		{"s settings", func(m *Model) bool { return m.settingsOpen() }},
	} {
		m := buildModel(t)
		m.layout.width, m.layout.height = 320, 40
		createSession(t, m, "alpha", t.TempDir(), "")
		if m = clickLabel(t, m, tc.label); !tc.open(m) {
			t.Errorf("click on %q opened nothing (err %q)", tc.label, m.errBar.text)
		}
	}
}

// A footer names keys the search field would take as typing, so the field
// keeps a click on them.
func TestClickOnListFooterWhileSearchingTypesNothing(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 200, 40
	x, y := paintedCell(t, m, "n new")
	m.rail.SetSearch(m.rail.Search(), true)
	preparedView(m)
	if m = leftClick(m, x, y); !m.onList() || !m.rail.Searching() || m.rail.Search() != "" {
		t.Fatalf("search owns the click, screen = %v searching = %v query = %q", m.screenID(), m.rail.Searching(), m.rail.Search())
	}
}

func TestClickOnConfirmFootAnswers(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	m.openConfirm(confirmTarget{prompt: uidialogs.Confirm{Action: uidialogs.ConfirmKill, Label: "kill alpha? frees its RAM."}, lifecycle: uilifecycle.Target{Action: uilifecycle.Action(uidialogs.ConfirmKill)}})
	if m = clickLabel(t, m, "n/esc cancel"); !m.onList() || pendingConfirm(m).prompt.Label != "" {
		t.Fatalf("cancel in the foot should close the confirm, screen = %v target = %+v", m.screenID(), pendingConfirm(m))
	}
}

func TestClickOnHelpFootCloses(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	m.openHelp()
	if m = clickLabel(t, m, "esc/q/? close"); !m.onList() {
		t.Fatalf("close in the foot should leave help, screen = %v", m.screenID())
	}
}

func TestClickOnMessagesFootRunsItsKeys(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	withNotice(m, strings.Repeat("a reason wide enough to give the card room for every key ", 2))
	m.openNotices("")

	for _, key := range []string{"↑↓ message", "pgup/pgdn"} {
		x, y := paintedCell(t, m, key)
		if _, ok := m.layout.keyHits.At(x, y); ok {
			t.Fatalf("%q moves the cursor; the rows and the wheel answer for it", key)
		}
	}
	if m = clickLabel(t, m, "esc"); !m.onList() {
		t.Fatalf("esc in the foot should close messages, screen = %v", m.screenID())
	}
}

// A key runs when the button comes back up over it, so whatever it opens
// starts after the release. A press that slides off the key runs nothing.
func TestAKeyRunsOnTheRelease(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 200, 40
	x, y := paintedCell(t, m, "? keys")
	if m = leftPress(m, x, y); !m.onList() {
		t.Fatalf("the press alone ran the key, screen = %v", m.screenID())
	}
	if m = leftRelease(m, 0, y); !m.onList() {
		t.Fatalf("a release off the key ran it, screen = %v", m.screenID())
	}
	if m = leftRelease(m, x, y); !m.onList() {
		t.Fatalf("a second release with no press ran the key, screen = %v", m.screenID())
	}
	if m = leftClick(m, x, y); !helpOpen(m) {
		t.Fatalf("a click on the key should run it, screen = %v", m.screenID())
	}
}

// A key repainted under a held button is not the key the press chose, so
// the release runs nothing.
func TestARepaintedKeyDoesNotRunOnTheRelease(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	m.layout.width, m.layout.height = 200, 40
	x, y := paintedCell(t, m, "? keys")
	m = leftPress(m, x, y)
	repainted := false
	for i, hit := range m.layout.keyHits {
		if hit.Y == y && x >= hit.X0 && x < hit.X1 {
			m.layout.keyHits[i].Key, repainted = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}, true
		}
	}
	if !repainted {
		t.Fatal("test setup: no key painted under the press")
	}
	if m = leftRelease(m, x, y); !m.onList() {
		t.Fatalf("the release ran the repainted key, screen = %v", m.screenID())
	}
}
