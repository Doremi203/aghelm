package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Doremi203/aghelm/internal/diff"
	"github.com/Doremi203/aghelm/internal/git"
	"github.com/Doremi203/aghelm/internal/ui/composer"
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uireview "github.com/Doremi203/aghelm/internal/ui/review"
	uisettings "github.com/Doremi203/aghelm/internal/ui/settings"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestSettingsToggleChromeIndependently(t *testing.T) {
	for _, tc := range []struct {
		name       string
		field      int
		hideHeader bool
		hideStats  bool
	}{
		{name: "header only", field: uisettings.FieldHeader, hideHeader: true},
		{name: "stats only", field: uisettings.FieldStats, hideStats: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := buildModel(t)
			m.openSettings()
			if m.settingsCard().State().HideHeader || m.settingsCard().State().HideStats {
				t.Fatal("header and stats should show by default")
			}
			settings := ansi.Strip(m.viewSettings())
			for _, want := range []string{"header", "computer stats"} {
				if !strings.Contains(settings, want) {
					t.Fatalf("settings missing %q:\n%s", want, settings)
				}
			}

			for m.settingsCard().State().Field != tc.field {
				m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyDown})
			}
			m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyRight})
			settings = ansi.Strip(m.viewSettings())
			headerValue, statsValue := "show", "show"
			if tc.hideHeader {
				headerValue = "hide"
			}
			if tc.hideStats {
				statsValue = "hide"
			}
			for _, row := range []struct {
				label string
				value string
			}{
				{label: "header", value: headerValue},
				{label: "computer stats", value: statsValue},
			} {
				found := false
				for _, line := range strings.Split(settings, "\n") {
					if !strings.Contains(line, row.label) {
						continue
					}
					found = true
					if !strings.Contains(line, row.value) {
						t.Fatalf("%s row missing %q:\n%s", row.label, row.value, line)
					}
				}
				if !found {
					t.Fatalf("settings missing %s row:\n%s", row.label, settings)
				}
			}
			m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyEnter})
			m.drainEffects(t)

			if m.prefs.hideHeader != tc.hideHeader || m.prefs.hideStats != tc.hideStats {
				t.Fatalf("model visibility = header %t stats %t, want header %t stats %t",
					m.prefs.hideHeader, m.prefs.hideStats, tc.hideHeader, tc.hideStats)
			}
			if storedFlag(t, m, uisettings.HideHeaderSetting) != tc.hideHeader || storedFlag(t, m, uisettings.HideStatsSetting) != tc.hideStats {
				t.Fatalf("reloaded visibility = header %t stats %t, want header %t stats %t",
					storedFlag(t, m, uisettings.HideHeaderSetting), storedFlag(t, m, uisettings.HideStatsSetting), tc.hideHeader, tc.hideStats)
			}
		})
	}
}

func TestListCanHideHeader(t *testing.T) {
	m := shotModel()
	shownBody := m.listBodyHeight()
	if got := m.headerRows(); got != 1 {
		t.Fatalf("shown header = %d rows, want 1", got)
	}

	m.prefs.hideHeader = true
	if got := m.headerRows(); got != 0 {
		t.Fatalf("hidden header = %d rows, want 0", got)
	}
	if rows := m.viewHeaderRows(); len(rows) != 0 {
		t.Fatalf("hidden header still paints %d rows", len(rows))
	}
	if got := m.listBodyHeight(); got != shownBody+1 {
		t.Fatalf("hidden header body = %d rows, want %d", got, shownBody+1)
	}
	if rows := strings.Split(preparedView(m), "\n"); len(rows) != m.layout.height {
		t.Fatalf("headerless frame = %d rows, terminal is %d", len(rows), m.layout.height)
	}
}

// The list frame is the rail alone and fills the terminal exactly.
func TestListFrameFillsTheTerminal(t *testing.T) {
	m := shotModel()
	frame := ansi.Strip(preparedView(m))
	if !strings.Contains(frame, "add-rate-limiting") {
		t.Fatalf("the frame lost the session tree:\n%s", frame)
	}
	rows := strings.Split(preparedView(m), "\n")
	if len(rows) != m.layout.height {
		t.Fatalf("the frame is %d rows, terminal is %d", len(rows), m.layout.height)
	}
	for _, row := range rows {
		if got := ansi.StringWidth(row); got > m.layout.width {
			t.Fatalf("a frame row is %d wide, terminal is %d", got, m.layout.width)
		}
	}
}

// Enter does a real tmux attach from the list.
func TestEnterAttaches(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	createSession(t, m, "handover", t.TempDir(), "")
	m.selectSessionRow(t, "handover")

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	*m = *updated.(*Model)
	if got := attachQueued(t, m); got == "" {
		t.Fatalf("enter did not queue an attach, err = %q", m.errBar.text)
	}
}

// A working session with nothing quotable yet animates a loader on its
// state line rather than holding a dash, and that loader keeps the
// startup tick alive so the frames actually advance.
func TestFullRowWorkingWithoutPaneLineAnimatesLoader(t *testing.T) {
	m := shotModel()
	m.prefs.comfortableRows = true
	m.workspace.paneLines = nil
	row := sessionRow(t, m, "add-rate-limiting")
	lines := uitheme.SplitLines(m.renderTreeRow(row, false, m.layout.width-1, 4, uitheme.PanelHex()))
	if len(lines) != 3 {
		t.Fatalf("comfortable row painted %d lines, want 3", len(lines))
	}
	frame := presentation.GlyphsFor("").StartupFrames[m.startup.LoadPhase%len(presentation.GlyphsFor("").StartupFrames)]
	state := ansi.Strip(lines[2])
	if !strings.Contains(state, frame+" working") {
		t.Fatalf("working row without a pane line should animate a loader, got %q", state)
	}
	if !m.needsLoaderTick() {
		t.Fatal("a loader row should keep the startup tick alive")
	}
	m.workspace.paneLines = map[string]string{"add-rate-limiting": "Running tests…", "ui-polish": "Compiling…"}
	m.rebuildRows()
	if m.rail.HasWorkingLoader() {
		t.Fatal("a quotable pane line should retire the loader")
	}
}

// Sessions are pinned to the whole terminal body, the size an attach gives
// them.
func TestPaneTargetSizeIsTheBody(t *testing.T) {
	m := shotModel()
	width, height := m.paneTargetSize()
	if width != m.layout.width {
		t.Fatalf("target width = %d, want the terminal's %d", width, m.layout.width)
	}
	if height != m.listBodyHeight() {
		t.Fatalf("target height = %d, want the body's %d", height, m.listBodyHeight())
	}
}

func TestQuickFooterLeavesThePaneSizeAlone(t *testing.T) {
	t.Parallel()
	m := buildModel(t)
	seedTwoGroups(t, m)
	setRailCursor(m, 1)
	preparedView(m)
	width, height := m.paneTargetSize()
	resting := m.listBodyHeight()

	m.openQuickMode()
	m = applyMsg(t, m, composer.PasteTextMsg{Target: composer.Quick, Gen: m.quick().Gen,
		Inner: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(strings.Repeat("filler word ", 40))}})
	preparedView(m)
	if w, h := m.paneTargetSize(); w != width || h != height {
		t.Fatalf("pane target moved from %dx%d to %dx%d", width, height, w, h)
	}
	if body := m.listBodyHeight(); body >= resting {
		t.Fatalf("the prompt's rows should come from the body: %d, resting %d", body, resting)
	}
}

func TestRailFootShowsBattery(t *testing.T) {
	m := shotModel()
	m.workspace.snap.BatteryOK, m.workspace.snap.BatteryPercent = true, 33
	foot := ansi.Strip(strings.Join(m.railFootLines(m.layout.width), "\n"))
	if !strings.Contains(foot, "batt 33%") || strings.Contains(foot, "charging") {
		t.Fatalf("foot lacks the battery reading:\n%s", foot)
	}
	m.workspace.snap.BatteryCharging = true
	foot = ansi.Strip(strings.Join(m.railFootLines(m.layout.width), "\n"))
	if !strings.Contains(foot, "batt 33% charging") {
		t.Fatalf("foot lacks the charging suffix:\n%s", foot)
	}
	m.workspace.snap.BatteryOK = false
	if foot = ansi.Strip(strings.Join(m.railFootLines(m.layout.width), "\n")); strings.Contains(foot, "batt") {
		t.Fatalf("foot shows a battery the machine lacks:\n%s", foot)
	}
}

func BenchmarkListFrame(b *testing.B) {
	m := shotModel()
	m.layout.width, m.layout.height = 200, 50
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = preparedView(m)
	}
}

func BenchmarkCloseLargeReview(b *testing.B) {
	m := shotModel()
	files := make([]diff.FileDiff, 10000)
	for i := range files {
		files[i].File = git.ChangedFile{Path: fmt.Sprintf("path/to/file-%05d.go", i)}
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m.base = baseReview
		seedReviewForTest(m, uireview.Target{ID: "bench"}, git.ScopeUncommitted, "/repo", diff.Set{Files: files}, true)
		b.StartTimer()
		m.closeDiff()
	}
}
