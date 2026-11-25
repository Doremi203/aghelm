package ui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Doremi203/aghelm/internal/diff"
	"github.com/Doremi203/aghelm/internal/git"
	"github.com/Doremi203/aghelm/internal/keybind"
	"github.com/Doremi203/aghelm/internal/store"
	uiframe "github.com/Doremi203/aghelm/internal/ui/frame"
	uinotices "github.com/Doremi203/aghelm/internal/ui/notices"
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uireview "github.com/Doremi203/aghelm/internal/ui/review"
	uisettings "github.com/Doremi203/aghelm/internal/ui/settings"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestViewPinsTheFrameToTheTerminalHeight(t *testing.T) {
	tall := "a\nb\nc\nd\ne"
	got := uiframe.Clamp(tall, 3, func() string { return uitheme.Paint("", 1, uitheme.BackdropHex()) })
	if got != "a\nb\nc" {
		t.Fatalf("clamp trim = %q", got)
	}
	short := "x"
	got = uiframe.Clamp(short, 3, func() string { return uitheme.Paint("", 1, uitheme.BackdropHex()) })
	lines := strings.Split(got, "\n")
	if len(lines) != 3 || lines[0] != "x" {
		t.Fatalf("clamp pad = %q", got)
	}
	for _, line := range lines[1:] {
		// Padding rows wear the backdrop fill rather than being bare, so
		// a short frame cannot show the terminal's background through its
		// bottom rows.
		if !strings.Contains(line, " ") {
			t.Fatalf("padding row is bare: %q", line)
		}
	}
}

// TestZZShot renders a full frame to disk for visual review. Skipped
// unless AM_SHOT names an output path.
func TestZZShot(t *testing.T) {
	out := os.Getenv("AM_SHOT")
	if out == "" {
		t.Skip("set AM_SHOT to render a frame")
	}
	if name := os.Getenv("AM_SHOT_THEME"); name != "" {
		applyTheme(uitheme.Palettes[uitheme.Index(name)])
		t.Cleanup(func() { applyTheme(uitheme.Palettes[0]) })
	}
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	m := shotModel()
	switch os.Getenv("AM_SHOT_MODE") {
	case "settings":
		m.push(newSettingsSurface(m, uisettings.State{
			ToolNames: []string{"claude", "codex", "grok", "opencode"},
			ToolIndex: 0, ThemeIndex: 1, Field: uisettings.FieldTheme, LayoutSplit: true,
		}))
		m.version = "0.9.2"
	case "help":
		m.openHelp()
	}
	if phase := os.Getenv("AM_SHOT_PHASE"); phase != "" {
		n, err := strconv.Atoi(phase)
		if err != nil {
			t.Fatal(err)
		}
		m.startup.BannerPhase = n
	}
	if err := os.WriteFile(out, []byte(preparedView(m)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A frame line wider than the terminal wraps, which pushes the whole
// layout down a row and tears the panels. A frame with more rows than the
// terminal loses its footer to the clamp. Both have to hold at every size
// the layout still makes sense at, including the width where the header
// swaps between the wordmark and its one-line form.
func TestFrameFitsTerminal(t *testing.T) {
	for _, width := range []int{80, 100, 120, 160, 200} {
		for _, height := range []int{24, 34, 50} {
			m := shotModel()
			m.layout.width, m.layout.height = width, height
			// View clamps and pads, which would hide a frame that is a row
			// short or a row long, so the raw frame is what gets measured.
			raw := strings.Split(m.viewListFrame(), "\n")
			if len(raw) != height {
				t.Errorf("%dx%d: frame paints %d rows", width, height, len(raw))
			}
			lines := strings.Split(preparedView(m), "\n")
			for i, line := range lines {
				if got := ansi.StringWidth(line); got > width {
					t.Errorf("%dx%d: line %d is %d wide: %q", width, height, i, got, ansi.Strip(line))
				}
			}
		}
	}
}

func TestViewDoesNotPublishCursorState(t *testing.T) {
	m := &Model{}
	m.layout.imeCursor = &uiframe.CursorAnchor{}
	m.layout.imeCursor.Set(8, 4, true)
	if got := m.View(); got != "loading..." {
		t.Fatalf("initial view = %q", got)
	}
	col, row, ok := m.layout.imeCursor.Get()
	if col != 8 || row != 4 || !ok {
		t.Fatalf("View changed published cursor to (%d, %d, %t)", col, row, ok)
	}
}

func TestUpdatePublishesPreparedFrame(t *testing.T) {
	m := shotModel()
	m.prepareFrame()
	before := m.View()
	m.openHelp()
	if got := m.View(); got != before {
		t.Fatal("View rendered state that had not been prepared")
	}
	updated, _ := m.Update(struct{}{})
	if updated != m {
		t.Fatal("Update replaced the root model")
	}
	if got := m.View(); got == before || got == "loading..." {
		t.Fatal("Update did not publish the changed frame")
	}
}

func preparedView(m *Model) string {
	m.prepareFrame()
	return m.View()
}

func TestHeaderShowsTheVersionBesideWordmark(t *testing.T) {
	m := &Model{
		layout:  layoutState{width: 120},
		version: "0.8.1",
	}
	header := ansi.Strip(m.viewHeaderRows()[0])
	if !strings.Contains(header, " v0.8.1") || strings.Index(header, "v0.8.1") > strings.Index(header, "agents") {
		t.Errorf("header should carry the running version by the wordmark: %q", header)
	}
	m.version = "dev"
	if header := ansi.Strip(m.viewHeaderRows()[0]); !strings.Contains(header, " dev") || strings.Contains(header, "vdev") {
		t.Errorf("dev build should read dev: %q", header)
	}
}

func TestViewLegendPaintsWaitingMessagesWarm(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	waiting := &Model{
		services: services{store: st, keys: keybind.DefaultSession(), listKeys: keybind.DefaultList()},
		version:  "0.40.0",
		notices:  noticesPanel{Panel: uinotices.NewPanel(nil, nil), Log: uinotices.Log{ImportError: "refused"}},
	}
	bar := paintLegend([]legendSection{waiting.viewLegend()}, 160, presentation.DefaultGlyphMode).Text
	if !strings.Contains(bar, uitheme.LegendAlertStyle().Render("M unread")) {
		t.Fatalf("waiting messages should use the working color:\n%s", bar)
	}
	idle := &Model{
		services: services{store: st, keys: keybind.DefaultSession(), listKeys: keybind.DefaultList()},
		version:  "0.40.0",
		notices:  noticesPanel{Panel: uinotices.NewPanel(nil, nil)},
	}
	idleBar := ansi.Strip(paintLegend([]legendSection{idle.viewLegend()}, 160, presentation.DefaultGlyphMode).Text)
	if strings.Contains(idleBar, "unread") {
		t.Fatalf("an empty panel should leave the messages key off:\n%s", idleBar)
	}
}

// The root's tier literals reach the painter: every field the row and view
// legends set survives the copy into ui/frame.
func TestRootLegendKeepsItsTiersAndHits(t *testing.T) {
	sections := []legendSection{{
		Title:  "Session",
		Leads:  [][2]string{{"p", "quick prompt mode"}},
		Alerts: [][2]string{{"M", "unread"}},
		Hint:   "a hint",
		Pairs:  [][2]string{{"r", "rename"}},
	}}
	painted := paintLegend(sections, 200, presentation.DefaultGlyphMode)
	plain := ansi.Strip(painted.Text)
	for _, want := range []string{"Session", "p quick prompt mode", "M unread", "a hint", "r rename"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("root legend lost %q:\n%s", want, plain)
		}
	}
	if len(painted.Hits) != 3 {
		t.Fatalf("hits = %+v, want one per pressable pair", painted.Hits)
	}
	lines := strings.Split(plain, "\n")
	for _, hit := range painted.Hits {
		if hit.Y < 0 || hit.Y >= len(lines) {
			t.Fatalf("hit %+v lands on no painted row", hit)
		}
	}
}

// The reviewed repo is code the user did not write. Our own rendering emits
// nothing but SGR colour sequences, so any other control byte in a painted
// frame is the repo driving the terminal.
func strayControl(frame string) string {
	if !utf8.ValidString(frame) {
		return "malformed UTF-8, where a raw 8-bit CSI or OSC would hide"
	}
	for i := 0; i < len(frame); i++ {
		c := frame[i]
		if c == '\n' || !presentation.IsControlByte(c) {
			continue
		}
		if n := sgrLen(frame[i:]); n > 0 {
			i += n - 1
			continue
		}
		start := max(0, i-20)
		return frame[start:min(len(frame), i+20)]
	}
	return ""
}

// sgrLen returns the length of the SGR colour sequence at the head of s, or 0
// when s does not start with one.
func sgrLen(s string) int {
	if len(s) < 3 || s[0] != 0x1b || s[1] != '[' {
		return 0
	}
	for i := 2; i < len(s); i++ {
		if s[i] == 'm' {
			return i + 1
		}
		if (s[i] < '0' || s[i] > '9') && s[i] != ';' && s[i] != ':' {
			return 0
		}
	}
	return 0
}

func TestEscapeControlsShowsBytesAndKeepsUTF8(t *testing.T) {
	cases := map[string]string{
		"\x1b]0;PWNED\x07": "^[]0;PWNED^G",
		"a\x1b[2Jb":        "a^[[2Jb",
		"over\rwrite":      "over^Mwrite",
		"del\x7f":          "del^?",
		"nul\x00":          "nul^@",
		"שלום ok":          "שלום ok",
		"plain":            "plain",
		// A lone 0x9b/0x9d is the 8-bit CSI/OSC and is malformed UTF-8.
		"x\x9b31mred":  `x\x9B31mred`,
		"x\x9d0;P\x9c": `x\x9D0;P\x9C`,
		// The same code points encoded properly are C1 controls all the
		// same: a terminal that reads them obeys them, and ansi.StringWidth
		// counts them as nothing while they take a cell.
		"x\u009b31m": `x\u009B31m`,
		"x\u009d0;P": `x\u009D0;P`,
	}
	for in, want := range cases {
		if got := presentation.EscapeControls(in); got != want {
			t.Errorf("presentation.EscapeControls(%q) = %q, want %q", in, got, want)
		}
	}
}

// A git error arrives from CombinedOutput with its lines intact, and the hint
// under the first line is the half that says what to do. Only a surface owning
// exactly one row folds them together.
func TestEscapeControlsKeepsNewlinesAndInlineDropsThem(t *testing.T) {
	const gitErr = "fatal: ambiguous argument 'x'\nUse '--' to separate paths from revisions"
	if got := presentation.EscapeControls(gitErr); got != gitErr {
		t.Errorf("presentation.EscapeControls(%q) = %q, want it untouched", gitErr, got)
	}
	if got := presentation.EscapeControls("a\n\x1bb"); got != "a\n^[b" {
		t.Errorf(`presentation.EscapeControls("a\n\x1bb") = %q, want "a\n^[b"`, got)
	}
	want := "fatal: ambiguous argument 'x' Use '--' to separate paths from revisions"
	if got := presentation.EscapeControlsInline(gitErr); got != want {
		t.Errorf("presentation.EscapeControlsInline(%q) = %q, want %q", gitErr, got, want)
	}
	if got := presentation.EscapeControlsInline("a\n\x1bb"); got != "a ^[b" {
		t.Errorf(`presentation.EscapeControlsInline("a\n\x1bb") = %q, want "a ^[b"`, got)
	}
}

// Hebrew encodes continuation bytes in the 0x80-0x9f range, so a byte-range
// check for C1 would mangle it. Escaping keys off malformed UTF-8 instead.
func TestEscapeControlsKeepsTextWhoseBytesLookLikeC1(t *testing.T) {
	for _, text := range []string{"שלום", "מה קורה", "日本語", "café"} {
		if got := presentation.EscapeControls(text); got != text {
			t.Errorf("presentation.EscapeControls(%q) = %q, want it untouched", text, got)
		}
	}
}

// The agent in the pane runs `aghelm rename` itself, so the session
// name is content the review header paints, not something the user typed.
func TestSessionNameCannotDriveTheTerminal(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	if m.services.gitDrv == nil {
		t.Skip("git not installed")
	}
	openReviewOn(t, m, "named", gitRepoWithTwoChangedFiles(t))
	m.layout.width, m.layout.height = 120, 40
	for i := range m.workspace.sessions {
		if m.workspace.sessions[i].ID == m.review.Snapshot().SessionID {
			m.workspace.sessions[i].Name = "rev\x1b]0;PWNED\x07iew"
		}
	}

	frame := m.viewDiffFull()
	if !strings.Contains(frame, "rev^[]0;PWNED^Giew") {
		t.Fatal("the session name should read as caret notation in the header")
	}
	if stray := strayControl(frame); stray != "" {
		t.Errorf("a session name leaks a control byte near %q", stray)
	}

	state := m.review.Snapshot()
	m.review.ApplySend(uireview.SendResult{TargetID: state.SessionID, RepoRoot: state.RepoSelected, Round: 1, Count: 1, TargetName: "rev\x1b]0;PWNED\x07iew", Outcome: uireview.SendConfirmed})
	if stray := strayControl(m.viewDiffFooter()); stray != "" {
		t.Errorf("the review notice leaks a control byte near %q", stray)
	}
	m.errBar.text = "rename failed for rev\x1b]0;PWNED\x07iew"
	if stray := strayControl(m.statusMessage(m.glyphs().ErroredGlyph, m.glyphs().FinishedGlyph, "▲")); stray != "" {
		t.Errorf("the status bar leaks a control byte near %q", stray)
	}
}

// git.Driver.run embeds CombinedOutput, and the second line of an ambiguous
// -argument error is the one that says what to do about it. The review's empty
// state owns the rows to show it; paint would truncate a folded single row.
func TestMultiLineGitErrorKeepsItsLines(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	req := seedReviewForTest(m, uireview.Target{ID: "s1", Cwd: "/repo"}, git.ScopeUncommitted, "/repo", diff.Set{}, false)
	m.review.ApplyLoad(uireview.LoadResult{TargetID: "s1", Scope: req.Scope, Generation: req.Generation, RepoRoot: "/repo", RepoRoots: []string{"/repo"}, Err: errors.New("fatal: ambiguous argument 'x'\nUse '--' to separate paths from revisions")})
	rows := strings.Split(m.review.EmptyText(m.reviewRenderContext()), "\n")
	if len(rows) != 2 {
		t.Fatalf("a two-line git error rendered %d rows: %q", len(rows), rows)
	}
	if !strings.Contains(rows[1], "separate paths from revisions") {
		t.Fatalf("the hint line was lost, rows = %q", rows)
	}
}

// The whole point: a file under review must not be able to drive the terminal
// the review is painted into. Both layouts, because they render through
// different cells.
func TestReviewedFileCannotDriveTheTerminal(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	if m.services.gitDrv == nil {
		t.Skip("git not installed")
	}
	// An OSC window-title set and a clear-screen, sequences our renderer
	// never emits, so finding them in the frame is unambiguous.
	payload := "const banner = \"\x1b]0;PWNED\x07\x1b[2J\x1b[31mgotcha\"\n"
	dir := gitRepoWithPayload(t, "payload.go", payload)
	openReviewOn(t, m, "escape", dir)
	m.layout.width, m.layout.height = 120, 40

	for _, layout := range []string{"unified", "split"} {
		frame := m.viewDiffFull()
		if !strings.Contains(frame, "PWNED") {
			t.Fatalf("%s: the payload line should be on screen to be a real test", layout)
		}
		if stray := strayControl(frame); stray != "" {
			t.Errorf("%s layout leaks a control byte to the terminal near %q", layout, stray)
		}
		m.review.ToggleSideBySide(m.diffCodeHeight())
	}
}

// A clean repo proves the assertion above is not vacuous: the frame really
// does carry nothing but SGR, so a stray control byte means something leaked.
func TestCleanReviewFrameCarriesOnlySGR(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	if m.services.gitDrv == nil {
		t.Skip("git not installed")
	}
	openReviewOn(t, m, "clean", gitRepoWithTwoChangedFiles(t))
	m.layout.width, m.layout.height = 120, 40
	if stray := strayControl(m.viewDiffFull()); stray != "" {
		t.Fatalf("a clean review frame should be SGR only, found %q", stray)
	}
}

// Git is driven with -z, so a path is handed to us as raw bytes and reaches
// the file rail, the code title and the comment bar unquoted.
func TestReviewedPathCannotDriveTheTerminal(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	if m.services.gitDrv == nil {
		t.Skip("git not installed")
	}
	dir := committedRepo(t)
	name := "w\x1b]0;P\x07.go"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("package a\n"), 0o644); err != nil {
		t.Skipf("filesystem rejects control bytes in a name: %v", err)
	}
	openReviewOn(t, m, "path", dir)
	m.layout.width, m.layout.height = 120, 40

	frame := m.viewDiffFull()
	if !strings.Contains(frame, "w^[]0;P^G.go") {
		t.Fatalf("the path should read as caret notation on screen, got %q", frame)
	}
	if stray := strayControl(frame); stray != "" {
		t.Errorf("a reviewed path leaks a control byte near %q", stray)
	}
}

// gitRepoWithPayload commits a plain first version of name, then writes the
// payload over it, so review has a real change to render.
func gitRepoWithPayload(t *testing.T, name, content string) string {
	t.Helper()
	dir := committedRepo(t)
	writeFile(t, dir, name, "package a\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "add")
	writeFile(t, dir, name, content)
	return dir
}

func committedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	writeFile(t, dir, "seed.go", "package a\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

// With nothing under the cursor there is nothing to act on, so the footer
// carries only the app-wide tier.
func TestFooterWithoutASelectedRow(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	resetRailModel(m)
	footer := ansi.Strip(m.viewFooter())
	if strings.Contains(footer, "Session") || strings.Contains(footer, "Group") {
		t.Fatalf("no row selected, no row tier:\n%s", footer)
	}
	if !strings.Contains(footer, "View") {
		t.Fatalf("the app-wide tier should stay:\n%s", footer)
	}
}

func TestFooterTierFollowsTheCursor(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "legend", t.TempDir(), "")
	m.applyCmd(t, m.refreshCmd())

	for i, row := range railRows(m) {
		if !row.isGroup {
			setRailCursor(m, i)
			break
		}
	}
	if footer := ansi.Strip(m.viewFooter()); !strings.Contains(footer, "Session") {
		t.Fatalf("a session under the cursor should title the tier Session:\n%s", footer)
	}

	for i, row := range railRows(m) {
		if row.isGroup {
			setRailCursor(m, i)
			break
		}
	}
	footer := ansi.Strip(m.viewFooter())
	if !strings.Contains(footer, "Root") {
		t.Fatalf("the root under the cursor should title the tier Root:\n%s", footer)
	}
	if strings.Contains(footer, "fork") {
		t.Fatalf("a group cannot be forked, so the key should not be offered:\n%s", footer)
	}
}

// A toggle's label names what the key will do next, not the state it is
// already in, so the footer reads as an instruction.
func TestFooterTogglesNameTheNextAction(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	// Wide enough that the row budget keeps every app-wide binding.
	m.layout.width = 260
	dir := t.TempDir()
	if err := m.services.store.AddGroup("work", dir, "off", "", ""); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	m.applyCmd(t, m.refreshCmd())
	createSession(t, m, "toggles", dir, "work")
	m.applyCmd(t, m.refreshCmd())

	if footer := m.viewFooter(); !strings.Contains(footer, uitheme.KeyCapQuiet("F", "fold all")) {
		t.Fatalf("an open tree should offer folding:\n%s", ansi.Strip(footer))
	}
	updated, _ := m.handleKey(runeKey("F"))
	m = updated.(*Model)
	if footer := m.viewFooter(); !strings.Contains(footer, uitheme.KeyCapQuiet("F", "unfold all")) {
		t.Fatalf("a folded tree should offer unfolding:\n%s", ansi.Strip(footer))
	}

	group := -1
	for i, row := range railRows(m) {
		if row.isGroup && !row.isRoot() {
			group = i
			break
		}
	}
	if group < 0 {
		t.Fatalf("the fixture should list a group to fold, rows: %v", m.groupRowPaths())
	}
	setRailCursor(m, group)
	if footer := m.viewFooter(); !strings.Contains(footer, uitheme.KeyCap("↵", "unfold")) {
		t.Fatalf("a collapsed group should offer unfolding:\n%s", ansi.Strip(footer))
	}

	m.rail.SetArchived(true)
	if footer := m.viewFooter(); !strings.Contains(footer, uitheme.KeyCapQuiet("t", "back to active")) {
		t.Fatalf("the archived view should offer the way back:\n%s", ansi.Strip(footer))
	}
}

func TestLegendOffersEmptyGroupsOnlyInTheActiveView(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	offered := func() bool {
		for _, pair := range m.viewLegend().Pairs {
			if pair[1] == "hide empty" || pair[1] == "show empty" {
				return true
			}
		}
		return false
	}
	if !offered() {
		t.Fatal("the active view should offer hiding empty groups")
	}
	m.rail.SetHideEmptyGroups(true)
	if !offered() {
		t.Fatal("the active view should offer showing empty groups again")
	}
	m.rail.SetArchived(true)
	if offered() {
		t.Fatal("the archived view should not offer the empty-groups key")
	}
}

func TestRowLegendDropsRestoreInActiveView(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	dir := t.TempDir()

	if err := m.services.store.CreateGroup("zone", ""); err != nil {
		t.Fatalf("create group: %v", err)
	}
	m.applyCmd(t, m.refreshCmd())
	createSession(t, m, "alpha", dir, "zone")

	m.selectSessionRow(t, "alpha")
	legend := m.rowLegend()
	for _, pair := range legend.Pairs {
		if strings.Contains(pair[1], "restore") {
			t.Fatalf("session legend in active view should not offer restore, got %+v", pair)
		}
	}
	if !slices.ContainsFunc(legend.Pairs, func(pair [2]string) bool { return pair[1] == "archive" }) {
		t.Fatal("session legend in active view should still offer archive")
	}

	m.selectGroupRow(t, "zone")
	legend = m.rowLegend()
	for _, pair := range legend.Pairs {
		if strings.Contains(pair[1], "restore") {
			t.Fatalf("group legend in active view should not offer restore, got %+v", pair)
		}
	}
	if !slices.ContainsFunc(legend.Pairs, func(pair [2]string) bool { return pair[1] == "archive" }) {
		t.Fatal("group legend in active view should still offer archive")
	}
}

// In the archived view, archiveSelected no-ops, so the legend should offer
// only restore, on both a session row and a group row.
func TestRowLegendDropsArchiveInArchivedView(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	dir := t.TempDir()

	if err := m.services.store.CreateGroup("zone", ""); err != nil {
		t.Fatalf("create group: %v", err)
	}
	m.applyCmd(t, m.refreshCmd())
	createSession(t, m, "alpha", dir, "zone")
	if err := m.services.store.SetArchived(m.sessionRows()[0].ID, true); err != nil {
		t.Fatalf("archive: %v", err)
	}

	m.rail.SetArchived(true)
	m.applyCmd(t, m.refreshCmd())

	m.selectSessionRow(t, "alpha")
	legend := m.rowLegend()
	for _, pair := range legend.Pairs {
		if strings.Contains(pair[1], "archive") {
			t.Fatalf("session legend in archived view should not offer archive, got %+v", pair)
		}
	}
	if !slices.ContainsFunc(legend.Pairs, func(pair [2]string) bool { return pair[1] == "restore" }) {
		t.Fatal("session legend in archived view should still offer restore")
	}

	m.selectGroupRow(t, "zone")
	legend = m.rowLegend()
	for _, pair := range legend.Pairs {
		if strings.Contains(pair[1], "archive") {
			t.Fatalf("group legend in archived view should not offer archive, got %+v", pair)
		}
	}
	if !slices.ContainsFunc(legend.Pairs, func(pair [2]string) bool { return pair[1] == "restore" }) {
		t.Fatal("group legend in archived view should still offer restore")
	}
}

func TestQuickPromptFooterKeys(t *testing.T) {
	m := shotModel()
	pushQuick(m)
	m.quick().Input = textarea.New()
	m.quick().SetTools([]string{"claude"}, 0)

	footerOne := m.viewFooter()
	if strings.Contains(footerOne, "shift+tab") || strings.Contains(footerOne, "previous tool") {
		t.Errorf("one tool enabled, footer shouldn't have shift+tab: %q", footerOne)
	}
	if !strings.Contains(footerOne, "tab tool") {
		t.Errorf("one tool enabled, missing tab pair: %q", footerOne)
	}

	m.quick().SetTools([]string{"claude", "codex"}, 0)
	footerTwo := m.viewFooter()
	if !strings.Contains(footerTwo, "shift+tab") || !strings.Contains(footerTwo, "previous tool") {
		t.Errorf("two tools enabled, missing shift+tab pair: %q", footerTwo)
	}
	if !strings.Contains(footerTwo, "tab tool") {
		t.Errorf("two tools enabled, missing tab pair: %q", footerTwo)
	}
	if !strings.Contains(footerTwo, "ctrl+t worktree") {
		t.Errorf("missing ctrl+t worktree pair: %q", footerTwo)
	}
}
