package frame

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// RowColumns lays a list row out as a name column and a right-aligned meta
// column, so status, tool and age line up down the list instead of ragging
// off the end of each name. Rows too narrow to split keep meta inline and
// let the caller's truncation decide what survives.
func RowColumns(lead, meta string, width int) string {
	if meta == "" {
		return lead
	}
	const gap = 2
	leadWidth := ansi.StringWidth(lead)
	metaWidth := ansi.StringWidth(meta)
	if width < 1 || leadWidth+gap+metaWidth > width {
		return lead + strings.Repeat(" ", gap) + meta
	}
	return lead + strings.Repeat(" ", width-leadWidth-metaWidth) + meta
}

func DisplayGroup(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

func DiskBytes(b uint64) string {
	return formatBytes(b, 1000)
}

func HumanBytes(b uint64) string {
	return formatBytes(b, 1024)
}

func formatBytes(b, unit uint64) string {
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ScrollWindow keeps the cursor visible inside a height-limited window of
// single-line rows, reserving one line for each overflow indicator.
func ScrollWindow(total, cursor, height int) (int, int) {
	if total <= height {
		return 0, total
	}
	visible := height - 2
	if visible < 1 {
		visible = 1
	}
	start := cursor - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	return start, start + visible
}

// LabelWidth is the column a row's quiet label is padded to, so the values
// beside it line up as one column.
const LabelWidth = 7

// FitColumns lays the richest pair of readings that fits: rights are tried
// from richest to plainest, and for each, the lefts in turn. Nothing fitting
// means the plainest left is trimmed to what the column has.
func FitColumns(lefts, rights []string, width int) string {
	for _, right := range rights {
		for _, left := range lefts {
			if ansi.StringWidth(left)+ansi.StringWidth(right)+2 <= width {
				return RowColumns(left, right, width)
			}
		}
	}
	last := lefts[len(lefts)-1]
	for _, right := range rights {
		if room := width - ansi.StringWidth(right) - 2; room >= 8 {
			return RowColumns(ansi.Truncate(last, room, "…"), right, width)
		}
	}
	return RowColumns(ansi.Truncate(last, max(width, 1), "…"), "", width)
}
