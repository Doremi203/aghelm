package ui

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Doremi203/aghelm/internal/execution"
	"github.com/Doremi203/aghelm/internal/keybind"
	"github.com/Doremi203/aghelm/internal/store"
	"github.com/Doremi203/aghelm/internal/sysstat"
	uieffects "github.com/Doremi203/aghelm/internal/ui/effects"
	uinotices "github.com/Doremi203/aghelm/internal/ui/notices"
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uiterminal "github.com/Doremi203/aghelm/internal/ui/terminal"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func noticeStore(t *testing.T) *store.Store {
	t.Helper()
	st := openTestStore(t, filepath.Join(t.TempDir(), "state.db"))
	t.Cleanup(func() { st.Close() })
	return st
}

// noticeModel is a model on st that lists noticeTip, the reason reading
// details.
func noticeModel(st *store.Store, version string) *Model {
	m := &Model{
		version: version,
		poller:  &poller{runner: execution.New(execution.Dependencies{Store: st}, execution.Options{})},
		services: services{
			store:    st,
			keys:     keybind.DefaultSession(),
			listKeys: keybind.DefaultList(),
		},
		notices: noticesPanel{
			Panel: uinotices.NewPanel(uinotices.Dismissed(st), uinotices.Seen(st)),
		},
	}
	withNotice(m, "details")
	return m
}

// noticeModalFloor is the panel's narrowest content column.
const noticeModalFloor = 62

func noticeIDs(notices []uinotices.Notice) []string {
	ids := make([]string, len(notices))
	for i, n := range notices {
		ids[i] = n.ID
	}
	return ids
}

func persistNoticeDismissal(t *testing.T, m *Model, id string) {
	t.Helper()
	m.dismissNotice(id)
	m.drainEffects(t)
}

func TestActiveNoticesListTheRefusedImport(t *testing.T) {
	m := noticeModel(noticeStore(t), "v0.2.0")
	if ids := noticeIDs(m.activeNotices()); !slices.Equal(ids, []string{noticeTip}) {
		t.Fatalf("a fresh store should list the refused import alone, got %v", ids)
	}
}

func TestDismissPersistsAcrossRestart(t *testing.T) {
	st := noticeStore(t)
	m := noticeModel(st, "v0.2.0")
	persistNoticeDismissal(t, m, noticeTip)
	if contains(noticeIDs(m.activeNotices()), noticeTip) {
		t.Fatal("dismissed notice still active")
	}

	reopened := noticeModel(st, "v0.2.0")
	if contains(noticeIDs(reopened.activeNotices()), noticeTip) {
		t.Fatal("dismissal did not survive restart")
	}
}

func TestConfigNotImportedNoticeCarriesTheReason(t *testing.T) {
	st := noticeStore(t)
	m := noticeModel(st, "v0.2.0")
	withNotice(m, "")
	if contains(noticeIDs(m.activeNotices()), uinotices.ConfigNotImported) {
		t.Fatal("an import that went through has nothing to report")
	}
	m.notices.ImportError = "config.toml: keybindings.session.detach needs at least one key"
	var refused uinotices.Notice
	for _, n := range m.activeNotices() {
		if n.ID == uinotices.ConfigNotImported {
			refused = n
		}
	}
	if refused.ID == "" {
		t.Fatalf("want %s among %v", uinotices.ConfigNotImported, noticeIDs(m.activeNotices()))
	}
	if !contains(refused.Body, m.notices.ImportError) {
		t.Fatalf("the notice should carry the reason: %q", refused.Body)
	}
	persistNoticeDismissal(t, m, uinotices.ConfigNotImported)
	reopened := noticeModel(st, "v0.2.0")
	reopened.notices.ImportError = m.notices.ImportError
	if contains(noticeIDs(reopened.activeNotices()), uinotices.ConfigNotImported) {
		t.Fatal("dismissal did not survive restart")
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func footModel(t *testing.T) *Model {
	t.Helper()
	m := noticeModel(noticeStore(t), "v0.2.0")
	m.workspace.snap = sysstat.Snapshot{
		CPUPercent: 42, CPUOK: true,
		MemPercent: 63, MemOK: true, MemUsed: 10 << 30, MemTotal: 16 << 30,
		DiskPercent: 71, DiskOK: true, DiskAvailable: 120 << 30,
	}
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func modalModel(t *testing.T) *Model {
	t.Helper()
	m := footModel(t)
	m.layout.width, m.layout.height = 100, 34
	m.openNotices("")
	return m
}

func TestOpenNoticesFromList(t *testing.T) {
	m := footModel(t)
	m.layout.width, m.layout.height = 100, 34
	m.base = baseList
	m.clearSurfaces()
	m.handleKey(key("M"))
	if !m.noticesOpen() {
		t.Fatalf("M should open the notices modal, screen = %v", m.screenID())
	}
}

func TestNoticesViewListsAndDetails(t *testing.T) {
	m := modalModel(t)
	frame := ansi.Strip(preparedView(m))
	for _, want := range []string{"messages", "Your config.toml was not carried", "details", "dismiss"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("modal missing %q:\n%s", want, frame)
		}
	}

	for _, terminal := range []int{100, 40} {
		m.layout.width = terminal
		frame := ansi.Strip(preparedView(m))
		var widths []int
		for _, line := range strings.Split(frame, "\n") {
			trimmed := strings.TrimRight(line, " ")
			if strings.ContainsAny(trimmed, "╭╰│") {
				widths = append(widths, len([]rune(trimmed)))
			}
		}
		if len(widths) == 0 {
			t.Fatalf("width %d: no frame rows found", terminal)
		}
		for i, width := range widths {
			if width != widths[0] {
				t.Fatalf("width %d: frame rows must align: row %d is %d, first is %d\n%s", terminal, i, width, widths[0], frame)
			}
		}
	}
}

func TestNoticesLongBodyWrapsFully(t *testing.T) {
	m := modalModel(t)
	withNotice(m, "Each group can choose inherit, on, or off for spawning sessions into their own git worktree, set when creating a group or editing it later; children inherit from their parents, and the global setting stays the root fallback.")
	m.openNotices(noticeTip)
	frame := ansi.Strip(preparedView(m))
	for _, line := range strings.Split(frame, "\n") {
		if !strings.Contains(line, "╭") {
			continue
		}
		if got := lipgloss.Width(strings.TrimSpace(line)); got <= noticeModalFloor+4 {
			t.Fatalf("wide terminal must widen the modal past %d, got %d", noticeModalFloor+4, got)
		}
		break
	}
	for _, word := range []string{"worktree,", "parents,", "fallback."} {
		if !strings.Contains(frame, word) {
			t.Fatalf("long body must wrap, %q missing:\n%s", word, frame)
		}
	}
	for _, line := range strings.Split(frame, "\n") {
		if got := lipgloss.Width(line); got > m.layout.width {
			t.Fatalf("line overflows terminal at %d: %q", got, line)
		}
	}
}

func TestNoticesShortTerminalKeepsFrameAndHint(t *testing.T) {
	m := modalModel(t)
	m.layout.width, m.layout.height = 30, 12
	withNotice(m, "Each group can choose inherit, on, or off for spawning sessions into their own git worktree, set when creating a group or editing it later.")
	m.openNotices(noticeTip)
	lines := strings.Split(ansi.Strip(preparedView(m)), "\n")
	if len(lines) > m.layout.height {
		t.Fatalf("frame must fit %d rows, got %d", m.layout.height, len(lines))
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "╰") {
		t.Fatalf("short terminal ate the modal's bottom border:\n%s", joined)
	}
	for _, hint := range []string{"↵ open", "x dismiss", "esc/q/M"} {
		if !strings.Contains(joined, hint) {
			t.Fatalf("short terminal ate %q:\n%s", hint, joined)
		}
	}
	if !strings.Contains(joined, "┃") {
		t.Fatalf("a clipped body must show its scrollbar:\n%s", joined)
	}
}

func TestNoticesBodyScrollIsBoundedAndVisible(t *testing.T) {
	m := modalModel(t)
	m.layout.width, m.layout.height = 70, 14
	withNotice(m, strings.Repeat("a reason long enough to scroll ", 40))
	m.openNotices(noticeTip)

	before := ansi.Strip(preparedView(m))
	if !strings.Contains(before, "┃") {
		t.Fatalf("a clipped body shows its scrollbar:\n%s", before)
	}
	m.handleKey(key("pgdown"))
	if noticesSurfaceOf(m).Panel().Offset() == 0 {
		t.Fatalf("page down did not move the summary:\n%s", ansi.Strip(preparedView(m)))
	}
	limit := noticesSurfaceOf(m).Panel().ScrollLimit(m.activeNotices(), m.noticesFrame())
	for i := 0; i < 20; i++ {
		m.handleKey(key("pgdown"))
	}
	if noticesSurfaceOf(m).Panel().Offset() != limit {
		t.Fatalf("scroll offset = %d, want bounded limit %d", noticesSurfaceOf(m).Panel().Offset(), limit)
	}
}

func TestNoticesDismissAdvancesAndCloses(t *testing.T) {
	m := modalModel(t)
	total := len(m.activeNotices())
	for i := 0; i < total; i++ {
		m.handleKey(key("x"))
	}
	if !m.onList() {
		t.Fatal("dismissing the last notice should close the modal")
	}
	if len(m.activeNotices()) != 0 {
		t.Fatal("all notices should be gone")
	}
}

func TestNoticesEnterOpensURL(t *testing.T) {
	m := modalModel(t)
	var opened string
	openBrowser = func(url string) error {
		opened = url
		return nil
	}
	t.Cleanup(func() { openBrowser = uiterminal.DefaultOpenBrowser })

	want := m.activeNotices()[noticesSurfaceOf(m).Panel().Cursor()].URL
	if want == "" {
		t.Fatal("selected notice needs a url for this test")
	}
	_, cmd := m.handleKey(key("enter"))
	if opened != "" {
		t.Fatal("browser started during Update")
	}
	m.applyCmd(t, cmd)
	if opened != want {
		t.Fatalf("opened %q, want %q", opened, want)
	}
}

func TestNoticesEscCloses(t *testing.T) {
	m := modalModel(t)
	m.handleKey(key("esc"))
	if !m.onList() {
		t.Fatal("esc should close the modal")
	}
	if len(m.activeNotices()) == 0 {
		t.Fatal("esc must not dismiss anything")
	}
}

// The first launch against a store opens the setup and records the
// version, on a build from source as on any other; the next launch opens
// nothing.
func TestStartupOpensTheSetupOnce(t *testing.T) {
	st := noticeStore(t)
	if err := st.SetSetting(lastSeenVersionSetting, ""); err != nil {
		t.Fatal(err)
	}
	m := noticeModel(st, "dev")
	m.openStartupGreeting()
	if !m.setupOpen() {
		t.Fatalf("first launch should open the setup, mode=%v", m.screenID())
	}
	if seen, _ := st.Setting(lastSeenVersionSetting); seen != "dev" {
		t.Fatalf("last seen version = %q, want dev", seen)
	}

	again := noticeModel(st, "dev")
	again.openStartupGreeting()
	if !again.listClear() {
		t.Fatalf("a later launch must not greet again, mode=%v", again.screenID())
	}
}

func TestNoticesTinyTerminalStaysInside(t *testing.T) {
	m := modalModel(t)
	for _, width := range []int{5, 12, 30} {
		m.layout.width, m.layout.height = width, 10
		for _, line := range strings.Split(ansi.Strip(preparedView(m)), "\n") {
			if got := lipgloss.Width(line); got > width {
				t.Fatalf("width %d: line overflows at %d: %q", width, got, line)
			}
		}
	}
}

func TestEmptyNoticesPanelStillOpens(t *testing.T) {
	m := footModel(t)
	m.layout.width, m.layout.height = 100, 34
	m.base = baseList
	m.clearSurfaces()
	for _, n := range m.activeNotices() {
		persistNoticeDismissal(t, m, n.ID)
	}
	if len(m.activeNotices()) != 0 {
		t.Fatal("test needs an empty panel")
	}

	m.handleKey(key("M"))
	if !m.noticesOpen() {
		t.Fatalf("M should open an empty panel, screen = %v", m.screenID())
	}
	frame := ansi.Strip(preparedView(m))
	for _, want := range []string{"nothing new", "esc/q/M"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("empty panel missing %q:\n%s", want, frame)
		}
	}
}

func TestDismissNoticeDeduplicatesPendingEffect(t *testing.T) {
	m := noticeModel(noticeStore(t), "v0.2.0")
	m.dismissNotice(noticeTip)
	m.dismissNotice(noticeTip)

	if len(m.effects.Main.Pending) != 1 {
		t.Fatalf("duplicate dismissal queued %d effects, want 1", len(m.effects.Main.Pending))
	}
	request, ok := m.effects.Main.Pending[0].Request.(uinotices.DismissRequest)
	if !ok || request.ID != noticeTip {
		t.Fatalf("queued request = %#v", m.effects.Main.Pending[0].Request)
	}
	m.drainEffects(t)
}

func TestDismissNoticeFailureRestoresNotice(t *testing.T) {
	st := noticeStore(t)
	if err := st.SetSetting(uinotices.DismissedSetting, `{broken`); err != nil {
		t.Fatalf("seed malformed setting: %v", err)
	}
	m := noticeModel(st, "v0.2.0")
	m.openNotices(noticeTip)

	_, _ = m.handleKey(key("x"))
	if contains(noticeIDs(m.activeNotices()), noticeTip) {
		t.Fatal("accepted dismissal stayed visible while its write was pending")
	}
	cmd := m.nextEffectCmd()
	if cmd == nil {
		t.Fatal("dismissal did not queue persistence")
	}
	m.applyCmd(t, cmd)

	if !contains(noticeIDs(m.activeNotices()), noticeTip) {
		t.Fatal("failed persistence did not restore the notice")
	}
	if !m.noticesOpen() || m.activeNotices()[noticesSurfaceOf(m).Panel().Cursor()].ID != noticeTip {
		t.Fatalf("failed dismissal did not restore its modal selection: screen = %v cursor=%d", m.screenID(), noticesSurfaceOf(m).Panel().Cursor())
	}
	if !strings.Contains(m.errBar.text, "decode setting") {
		t.Fatalf("failure reason = %q", m.errBar.text)
	}
}

func TestDismissNoticeFailureDoesNotReopenAfterNewerInput(t *testing.T) {
	st := noticeStore(t)
	if err := st.SetSetting(uinotices.DismissedSetting, `{broken`); err != nil {
		t.Fatalf("seed malformed setting: %v", err)
	}
	m := noticeModel(st, "v0.2.0")
	m.openNotices(noticeTip)
	_, _ = m.handleKey(key("x"))
	cmd := m.nextEffectCmd()

	updated, _ := m.handleKey(key("esc"))
	m = updated.(*Model)
	m.applyCmd(t, cmd)

	if !m.onList() {
		t.Fatalf("failed old dismissal stole focus from newer input: screen = %v", m.screenID())
	}
	if !contains(noticeIDs(m.activeNotices()), noticeTip) {
		t.Fatal("failed old dismissal did not restore the notice card")
	}
}

func TestDismissNoticeFailureDuringQuitRestoresWithoutReopening(t *testing.T) {
	st := noticeStore(t)
	if err := st.SetSetting(uinotices.DismissedSetting, `{broken`); err != nil {
		t.Fatalf("seed malformed setting: %v", err)
	}
	m := noticeModel(st, "v0.2.0")
	// Leave one visible card so accepting its dismissal closes the modal.
	m.openNotices(noticeTip)

	_, _ = m.handleKey(key("x"))
	cmd := m.nextEffectCmd()
	if cmd == nil || !m.onList() {
		t.Fatalf("last dismissal was not accepted: cmd=%v screen = %v", cmd != nil, m.screenID())
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(*Model)
	if !m.effects.Quitting {
		t.Fatal("ctrl+c did not start draining the accepted dismissal")
	}
	m.applyCmd(t, cmd)

	if !m.onList() {
		t.Fatalf("failed dismissal reopened over quit: screen = %v", m.screenID())
	}
	if !contains(noticeIDs(m.activeNotices()), noticeTip) {
		t.Fatal("failed dismissal did not restore the notice data")
	}
	if !strings.Contains(m.errBar.text, "decode setting") {
		t.Fatalf("failure reason = %q", m.errBar.text)
	}
}

func TestNoticeMarksAreResolvedWithoutChangingExternalContent(t *testing.T) {
	m := noticeModel(noticeStore(t), "v1.0.0")
	const text = "⚙❯✳ literal reason"
	withNotice(m, text)
	for _, tc := range []struct {
		mode               presentation.GlyphMode
		kind, link, cursor string
	}{
		{presentation.GlyphModeBasic, "!", "|   enter", "|"},
		{presentation.GlyphModeAdvanced, "▲", "▍   ↵", "▍"},
	} {
		m.prefs.glyphMode = tc.mode
		notices := m.activeNotices()
		if !contains(notices[0].Body, text) {
			t.Fatal("the reason was rewritten")
		}
		m.layout.width, m.layout.height = 120, 40
		m.notices.Open(notices, noticeTip)
		got := ansi.Strip(strings.Join(m.notices.View(notices, m.noticesFrame()), "\n"))
		for _, want := range []string{tc.cursor + " " + tc.kind + " " + notices[0].Title[:20], text, tc.link + "    Read what Settings holds", "github.com"} {
			if !strings.Contains(got, want) {
				t.Fatalf("%s panel missing %q: %s", tc.mode, want, got)
			}
		}
	}
}

func scrollModel(t *testing.T, lines int) *Model {
	t.Helper()
	m := modalModel(t)
	m.layout.width, m.layout.height = 70, 14
	words := make([]string, lines)
	for index := range words {
		words[index] = fmt.Sprintf("change-line-%02d-padded-to-wrap-alone-on-its-row", index)
	}
	withNotice(m, strings.Join(words, " "))
	m.openNotices(noticeTip)
	return m
}

func thumbRows(frame string) []int {
	var rows []int
	for index, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, "┃") {
			rows = append(rows, index)
		}
	}
	return rows
}

func TestShortBodyHasNoScrollbar(t *testing.T) {
	m := scrollModel(t, 2)
	m.layout.height = 40
	if frame := ansi.Strip(preparedView(m)); strings.Contains(frame, "┃") {
		t.Fatalf("a body that fits needs no scrollbar:\n%s", frame)
	}
}

func TestWheelScrollsTheMessageBody(t *testing.T) {
	m := scrollModel(t, 30)
	before := thumbRows(ansi.Strip(preparedView(m)))
	m = wheelOnCard(m, tea.MouseButtonWheelDown)
	if noticesSurfaceOf(m).Panel().Offset() != uinotices.WheelRows {
		t.Fatalf("wheel down moved %d rows, want %d", noticesSurfaceOf(m).Panel().Offset(), uinotices.WheelRows)
	}
	for i := 0; i < 40; i++ {
		m = wheelOnCard(m, tea.MouseButtonWheelDown)
	}
	limit := noticesSurfaceOf(m).Panel().ScrollLimit(m.activeNotices(), m.noticesFrame())
	if noticesSurfaceOf(m).Panel().Offset() != limit {
		t.Fatalf("wheel scrolled to %d, want it bounded at %d", noticesSurfaceOf(m).Panel().Offset(), limit)
	}
	after := thumbRows(ansi.Strip(preparedView(m)))
	if len(before) == 0 || len(after) == 0 || after[0] <= before[0] {
		t.Fatalf("the thumb should have moved down: before %v, after %v", before, after)
	}
	m = wheelOnCard(m, tea.MouseButtonWheelUp)
	if noticesSurfaceOf(m).Panel().Offset() != limit-uinotices.WheelRows {
		t.Fatalf("wheel up moved to %d, want %d", noticesSurfaceOf(m).Panel().Offset(), limit-uinotices.WheelRows)
	}
	if !m.noticesOpen() {
		t.Fatalf("the wheel must not leave the panel, screen = %v", m.screenID())
	}
}

func TestHomeAndEndJumpTheMessageBody(t *testing.T) {
	m := scrollModel(t, 30)
	limit := noticesSurfaceOf(m).Panel().ScrollLimit(m.activeNotices(), m.noticesFrame())
	if limit <= 0 {
		t.Fatalf("the body should scroll, limit %d", limit)
	}
	for _, jump := range []string{"end", "G"} {
		m.handleKey(key("home"))
		m.handleKey(key(jump))
		if noticesSurfaceOf(m).Panel().Offset() != limit {
			t.Fatalf("%s moved to %d, want the bottom at %d", jump, noticesSurfaceOf(m).Panel().Offset(), limit)
		}
	}
	for _, jump := range []string{"home", "g"} {
		m.handleKey(key("end"))
		m.handleKey(key(jump))
		if noticesSurfaceOf(m).Panel().Offset() != 0 {
			t.Fatalf("%s moved to %d, want the top", jump, noticesSurfaceOf(m).Panel().Offset())
		}
	}
}

func TestGrowingTheTerminalKeepsTheLastPageInView(t *testing.T) {
	m := scrollModel(t, 30)
	m.handleKey(key("end"))
	bottom := noticesSurfaceOf(m).Panel().Offset()
	m.layout.height = 20
	frame := ansi.Strip(preparedView(m))
	if !strings.Contains(frame, "Read what Settings holds") || !strings.Contains(frame, "╰") {
		t.Fatalf("an offset past the new last page shows that last page inside the frame:\n%s", frame)
	}
	limit := noticesSurfaceOf(m).Panel().ScrollLimit(m.activeNotices(), m.noticesFrame())
	if limit >= bottom {
		t.Fatalf("the taller terminal should have fewer pages: limit %d, saved offset %d", limit, bottom)
	}
	m = wheelOnCard(m, tea.MouseButtonWheelUp)
	if noticesSurfaceOf(m).Panel().Offset() != limit-uinotices.WheelRows {
		t.Fatalf("the first scroll up starts from the page on screen: offset %d, want %d", noticesSurfaceOf(m).Panel().Offset(), limit-uinotices.WheelRows)
	}
}

func TestHintNamesHomeAndEndWhenItFits(t *testing.T) {
	m := modalModel(t)
	withNotice(m, strings.Repeat("wide words ", 12))
	m.openNotices(noticeTip)
	m.layout.width = 180
	if frame := ansi.Strip(preparedView(m)); !strings.Contains(frame, "pgup/pgdn/home/end scroll") {
		t.Fatalf("a wide frame names home and end:\n%s", frame)
	}
	m.layout.width = 70
	frame := ansi.Strip(preparedView(m))
	for _, want := range []string{"x dismiss", "esc"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("a narrow frame keeps %q:\n%s", want, frame)
		}
	}
	if strings.Contains(frame, "home/end") {
		t.Fatalf("a narrow frame keeps the short hint:\n%s", frame)
	}
}

func noticesOverList(t *testing.T) *Model {
	t.Helper()
	m := buildModel(t)
	m.layout.width, m.layout.height = 120, 34
	// The real trigger: the messages key on the list.
	m.Update(runeKey("M"))
	if !m.noticesOpen() {
		t.Fatal("M did not open the messages modal")
	}
	return m
}

func TestNoticesSurfaceOpensOverTheList(t *testing.T) {
	m := noticesOverList(t)
	if !m.onList() || len(m.surfaces) != 1 {
		t.Fatalf("screen = %v, surfaces = %d", m.screenID(), len(m.surfaces))
	}
	m.openNotices("")
	if len(m.surfaces) != 1 {
		t.Fatalf("opening an open panel stacked another: %d surfaces", len(m.surfaces))
	}
}

func TestNoticesSurfaceClosesOnEscAndBackdrop(t *testing.T) {
	if !(&uinotices.Surface{}).Dismissible() {
		t.Fatal("the messages modal must be dismissible")
	}
	for name, closeIt := range map[string]func(*Model) *Model{
		"esc": func(m *Model) *Model { m.Update(namedKey(tea.KeyEsc)); return m },
		"click off the card": func(m *Model) *Model {
			return clickBackdrop(t, m)
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := closeIt(noticesOverList(t))
			if m.noticesOpen() || m.effects.Quitting {
				t.Fatalf("modal open = %t, quitting = %t", m.noticesOpen(), m.effects.Quitting)
			}
		})
	}
}

// A dismissal that fails to persist reopens the modal only while nothing
// else is on screen; a card the user opened since owns the screen.
func TestNoticesSurfaceFencesAStaleDismissalCompletion(t *testing.T) {
	m := noticeModel(noticeStore(t), "v0.2.0")
	m.layout.width, m.layout.height = 120, 34
	m.openNotices(noticeTip)
	job := &uieffects.Job{Request: uinotices.DismissRequest{ID: noticeTip, ForegroundGen: m.gens.foreground, Modal: true}}

	m.Pop()
	m.openHelp()
	m.applyNoticeDismiss(job, errTest("persist failed"))
	if m.noticesOpen() {
		t.Fatal("a failed dismissal opened the modal over another card")
	}

	m.Pop()
	m.applyNoticeDismiss(job, errTest("persist failed"))
	if !m.noticesOpen() {
		t.Fatal("a failed dismissal did not reopen the modal on a clear list")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestNoticesCloseFollowsSavedBinding(t *testing.T) {
	m := noticeModel(noticeStore(t), "v0.2.0")
	m.openNotices("")
	m.services.listKeys = m.services.listKeys.With(keybind.Messages, bindingOf(t, "N", "f2"))
	m.handleKey(key("M"))
	if !m.noticesOpen() {
		t.Fatal("old messages binding closed dialog")
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyF2})
	if m.noticesOpen() {
		t.Fatal("saved messages binding did not close dialog")
	}
	m.openNotices("")
	m.services.listKeys = m.services.listKeys.With(keybind.Messages, bindingOf(t))
	m.handleKey(key("M"))
	if !m.noticesOpen() {
		t.Fatal("disabled messages binding retained old alias")
	}
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.noticesOpen() {
		t.Fatal("Escape did not close messages")
	}
}

func TestNoticesToggleDoesNotOverrideDismiss(t *testing.T) {
	m := noticeModel(noticeStore(t), "v0.2.0")
	m.openNotices("")
	m.services.listKeys = m.services.listKeys.With(keybind.Messages, bindingOf(t, "x"))
	m.handleKey(key("x"))
	m.drainEffects(t)
	if !m.notices.Dismissed(noticeTip) {
		t.Fatal("toggle stole the local dismiss action")
	}
}

// seenModel lists noticeTip on a terminal the panel fits.
func seenModel(st *store.Store) *Model {
	m := noticeModel(st, "v0.2.0")
	m.layout.width, m.layout.height = 100, 34
	return m
}

func unreadIDs(notices []uinotices.Notice) []string {
	var ids []string
	for _, n := range notices {
		if n.Unread {
			ids = append(ids, n.ID)
		}
	}
	return ids
}

func TestOpeningThePanelMarksTheSelectedNoticeSeen(t *testing.T) {
	st := noticeStore(t)
	m := seenModel(st)
	if got := unreadIDs(m.activeNotices()); !slices.Equal(got, []string{noticeTip}) {
		t.Fatalf("a fresh store should leave the notice unread, got %v", got)
	}
	m.openNotices(noticeTip)
	m.drainEffects(t)
	if got := unreadIDs(m.activeNotices()); got != nil {
		t.Fatalf("unread after opening on the notice = %v, want none", got)
	}
	if raw, err := st.Setting(uinotices.SeenSetting); err != nil || raw != `["`+noticeTip+`"]` {
		t.Fatalf("seen setting = %q, %v", raw, err)
	}
}

func TestFooterChipSaysMessagesOnceEverythingIsSeen(t *testing.T) {
	m := seenModel(noticeStore(t))
	if _, label := m.messagesAlert(); label != "unread" {
		t.Fatalf("an unseen notice: chip = %q, want unread", label)
	}
	m.openNotices(noticeTip)
	m.Pop()
	m.drainEffects(t)
	if _, label := m.messagesAlert(); label != "messages" {
		t.Fatalf("every active notice seen: chip = %q, want messages", label)
	}
	persistNoticeDismissal(t, m, noticeTip)
	if glyph, label := m.messagesAlert(); glyph != "" || label != "" {
		t.Fatalf("nothing active: chip = %q %q, want none", glyph, label)
	}
}

func TestReadMessagesChipIsADimPair(t *testing.T) {
	m := seenModel(noticeStore(t))
	m.openNotices(noticeTip)
	m.Pop()
	m.drainEffects(t)
	legend := m.viewLegend()
	if legend.Alerts != nil {
		t.Fatalf("a read inbox raised the loud alert: %v", legend.Alerts)
	}
	if len(legend.Pairs) == 0 || legend.Pairs[0][1] != "messages" {
		t.Fatalf("the read chip should lead the quiet pairs, got %v", legend.Pairs)
	}
}

func TestSeenSurvivesARestart(t *testing.T) {
	st := noticeStore(t)
	m := seenModel(st)
	m.openNotices(noticeTip)
	m.drainEffects(t)

	reopened := seenModel(st)
	if got := unreadIDs(reopened.activeNotices()); got != nil {
		t.Fatalf("unread after a restart = %v, want none", got)
	}
}

func TestSeenWriteFailureKeepsTheMark(t *testing.T) {
	st := noticeStore(t)
	if err := st.SetSetting(uinotices.SeenSetting, `{broken`); err != nil {
		t.Fatalf("seed malformed setting: %v", err)
	}
	m := seenModel(st)
	m.openNotices(noticeTip)
	m.drainEffects(t)
	if !strings.HasPrefix(m.errBar.text, "mark message read: ") || !strings.Contains(m.errBar.text, "decode setting") {
		t.Fatalf("status bar = %q", m.errBar.text)
	}
	if got := unreadIDs(m.activeNotices()); got != nil {
		t.Fatalf("a failed write dropped the in-memory mark: unread = %v", got)
	}
}
