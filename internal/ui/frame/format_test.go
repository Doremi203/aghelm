package frame

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRowColumnsRightAlignsMeta(t *testing.T) {
	if got := RowColumns("api", "working", 20); got != "api"+strings.Repeat(" ", 10)+"working" {
		t.Fatalf("RowColumns = %q", got)
	}
	if got := RowColumns("api", "", 20); got != "api" {
		t.Fatalf("a row with no meta changed to %q", got)
	}
	if got := RowColumns("api", "working", 6); got != "api  working" {
		t.Fatalf("a narrow row should keep meta inline, got %q", got)
	}
}

func TestDisplayGroupNamesTheRoot(t *testing.T) {
	if got := DisplayGroup(""); got != "root" {
		t.Fatalf("empty group = %q, want root", got)
	}
	if got := DisplayGroup("work"); got != "work" {
		t.Fatalf("named group = %q", got)
	}
}

func TestHumanAndDiskBytesPickTheirBase(t *testing.T) {
	for _, tc := range []struct {
		value uint64
		want  string
	}{
		{0, "0B"},
		{999, "999B"},
		{1000, "1.0KB"},
		{1_000_000, "1.0MB"},
		{169_850_000_000, "169.8GB"},
		{1_000_000_000_000, "1.0TB"},
	} {
		if got := DiskBytes(tc.value); got != tc.want {
			t.Errorf("DiskBytes(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}
	if got := HumanBytes(1 << 20); got != "1.0MB" {
		t.Errorf("HumanBytes(1MiB) = %q, want 1.0MB", got)
	}
}

func TestScrollWindowKeepsTheCursorVisible(t *testing.T) {
	cases := []struct {
		name                  string
		total, cursor, height int
		wantStart, wantEnd    int
	}{
		{"fits entirely", 5, 2, 10, 0, 5},
		{"cursor at top", 100, 0, 20, 0, 18},
		{"cursor centered", 100, 50, 20, 41, 59},
		{"cursor at bottom", 100, 99, 20, 82, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := ScrollWindow(tc.total, tc.cursor, tc.height)
			if start != tc.wantStart || end != tc.wantEnd {
				t.Fatalf("ScrollWindow(%d,%d,%d) = %d,%d want %d,%d",
					tc.total, tc.cursor, tc.height, start, end, tc.wantStart, tc.wantEnd)
			}
			if tc.cursor < start || tc.cursor >= end {
				t.Fatalf("cursor %d outside window [%d,%d)", tc.cursor, start, end)
			}
		})
	}
}

// Nothing fitting whole means the plainest left keeps whatever readable room
// is left for it.
func TestFitColumnsTrimsThePlainestLeft(t *testing.T) {
	got := ansi.Strip(FitColumns([]string{"a very long name", "name"}, []string{"working · 3m"}, 20))
	if ansi.StringWidth(got) != 20 {
		t.Fatalf("fitted row is %d wide, want the column's 20: %q", ansi.StringWidth(got), got)
	}
	if !strings.Contains(got, "working") {
		t.Fatalf("the reading was dropped instead of the name trimmed: %q", got)
	}
}

func TestFitColumnsKeepsTheRichestPairThatFits(t *testing.T) {
	got := ansi.Strip(FitColumns([]string{"rich name", "name"}, []string{"state", ""}, 24))
	if !strings.Contains(got, "rich name") || !strings.Contains(got, "state") {
		t.Fatalf("fit = %q, want both the rich name and the reading", got)
	}
}
