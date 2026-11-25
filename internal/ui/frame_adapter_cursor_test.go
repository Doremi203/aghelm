package ui

import (
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	"strings"
	"testing"

	"github.com/Doremi203/aghelm/internal/keybind"
	uiframe "github.com/Doremi203/aghelm/internal/ui/frame"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestCustomSearchCursorsEmitMarkersOnlyWhileTyping(t *testing.T) {
	m := &Model{
		layout: layoutState{
			width:  100,
			height: 28,
		},
		services: services{
			keys:     keybind.DefaultSession(),
			listKeys: keybind.DefaultList(),
		},
		rail: railModelSearching(nil, 0, "中文"),
	}
	m.openHelp()
	m.handleKey(runeKey("/"))
	for _, r := range "定位" {
		m.handleKey(runeKey(string(r)))
	}
	if line := m.searchFieldLine(40); !strings.Contains(line, uiframe.CursorMarker) {
		t.Fatal("list search cursor has no marker")
	}
	if frame := viewHelpOf(m); !strings.Contains(frame, uiframe.CursorMarker) {
		t.Fatal("help search cursor has no marker")
	}

	m.rail.SetSearch(m.rail.Search(), false)
	m.handleKey(namedKey(tea.KeyEnter))
	if line := m.searchFieldLine(40); strings.Contains(line, uiframe.CursorMarker) {
		t.Fatal("closed list search kept a cursor marker")
	}
	if frame := viewHelpOf(m); strings.Contains(frame, uiframe.CursorMarker) {
		t.Fatal("closed help search kept a cursor marker")
	}
}

func TestNoActiveInputClearsCursorAnchor(t *testing.T) {
	m := &Model{
		layout: layoutState{imeCursor: &uiframe.CursorAnchor{}},
	}
	m.layout.imeCursor.Set(9, 7, true)
	if got := m.syncCursorAnchor("plain frame"); got != "plain frame" {
		t.Fatalf("frame changed to %q", got)
	}
	if _, _, ok := m.layout.imeCursor.Get(); ok {
		t.Fatal("inactive frame kept the previous cursor anchor")
	}
}

func TestFinalHelpLayoutPublishesAndRemovesCursorMarker(t *testing.T) {
	for _, mode := range []presentation.GlyphMode{presentation.GlyphModeBasic, presentation.GlyphModeAdvanced} {
		m := &Model{
			prefs: preferences{glyphMode: mode},
			layout: layoutState{
				width:     100,
				height:    28,
				imeCursor: &uiframe.CursorAnchor{},
			},
			services: services{
				keys:     keybind.DefaultSession(),
				listKeys: keybind.DefaultList(),
			},
		}
		m.openHelp()
		m.handleKey(runeKey("/"))
		for _, r := range "中文" {
			m.handleKey(runeKey(string(r)))
		}
		frame := preparedView(m)
		if strings.Contains(frame, uiframe.CursorMarker) {
			t.Fatal("private cursor marker leaked from the final frame")
		}
		col, row, ok := m.layout.imeCursor.Get()
		if !ok || col < 1 || col > m.layout.width || row < 1 || row > m.layout.height {
			t.Fatalf("final cursor anchor = (%d, %d, %v), frame is %dx%d", col, row, ok, m.layout.width, m.layout.height)
		}
		plain := strings.Split(ansi.Strip(frame), "\n")
		if row > len(plain) || !strings.Contains(plain[row-1], "search 中文"+presentation.ChromeFor(mode).CursorGlyph) {
			t.Fatalf("anchor row %d does not contain the help input: %q", row, plain[row-1])
		}
	}
}
