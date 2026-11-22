package notices

import (
	"net/url"
	"strings"
	"time"

	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var kindLabels = [...]string{KindNews: "NEWS", KindHeadsUp: "HEADS UP", KindTip: "TIP"}

// displayZone is where an expiry date is read: the viewer's own clock.
var displayZone = time.Local

// paneHit is the link bar the pane painted: a click on its line between x0
// and x1 opens the link.
type paneHit struct {
	line, x0, x1 int
}

// messagePane is a notice: its kind, banner and expiry, the title, the
// body, and the link it opens when it has one.
func messagePane(n Notice, width int, t Theme, mode presentation.GlyphMode) []string {
	head := chip(kindLabels[n.Kind], kindColor(n.Kind, t), t) + "  " + t.Muted.Render(n.Subtitle)
	if expires := expiry(n); expires != "" {
		head = atRight(ansi.Truncate(head, max(width-ansi.StringWidth(expires)-2, 0), "…"), t.Muted.Render(expires), width)
	}
	rows := []string{ansi.Truncate(head, width, "…"), ""}
	rows = appendWrapped(rows, []textRun{{text: n.Title}}, width, headingStyle(t), t)
	var body []string
	for _, line := range n.Body {
		body = appendWrapped(body, phraseRuns(line, n.Accent), width, t.Value, t)
	}
	if len(body) > 0 {
		rows = spaced(rows, body...)
	}
	if n.URL == "" {
		return rows
	}
	action := n.Action
	if action == "" {
		action = "Open link"
	}
	bar, _ := actionBar("↵", action, t.Muted.Render(linkHost(n.URL)), kindColor(n.Kind, t), width, t, mode)
	return spaced(rows, bar)
}

// linkHost is the host a link leads to, or the whole URL when it does not parse.
func linkHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return parsed.Host
}

func expiry(n Notice) string {
	if n.Expires.IsZero() {
		return ""
	}
	return "until " + n.Expires.In(displayZone).Format("Jan 2")
}

// measure is a notice's rows as they would read unwrapped, for the modal's
// width.
func measure(n Notice) []string {
	return append([]string{" " + kindLabels[n.Kind] + "   " + n.Subtitle + "  " + expiry(n), n.Title}, n.Body...)
}

// atRight sets right at the row's right edge, when two spaces still fit
// before it.
func atRight(row, right string, width int) string {
	if gap := width - lipgloss.Width(row) - lipgloss.Width(right); gap >= 2 {
		return row + strings.Repeat(" ", gap) + right
	}
	return row
}

// chip is a label on a fill, in whichever of the backdrop's ink and the bright text reads clearer on it.
func chip(text string, fill lipgloss.Color, t Theme) string {
	return lipgloss.NewStyle().Background(fill).Foreground(chipInk(fill, t)).Bold(true).Render(" " + text + " ")
}

func chipInk(fill lipgloss.Color, t Theme) lipgloss.Color {
	if uitheme.ContrastRatio(string(t.Bright), string(fill)) > uitheme.ContrastRatio(string(t.Ink), string(fill)) {
		return t.Bright
	}
	return t.Ink
}

// actionBar is a call to action across the pane on a fill tinted toward
// tone: a bar and the key in that tone, what it does, and at the right
// where it leads. cells is how far the key and what it does reach.
func actionBar(key, label, right string, tone lipgloss.Color, width int, t Theme, mode presentation.GlyphMode) (row string, cells int) {
	bar := lipgloss.NewStyle().Foreground(tone).Render(presentation.GlyphsFor(mode).SectionBarGlyph)
	row = bar + " " + chip(" "+presentation.KeyLabel(mode, key)+" ", tone, t) + "  " + headingStyle(t).Render(label)
	cells = lipgloss.Width(row)
	if right != "" {
		row = atRight(row, ansi.Truncate(right, max(width-cells-3, 0), "…"), width-1)
	}
	return t.Callout(row, width, tone), cells
}

func groupLabel(text string, t Theme) string { return t.Muted.Bold(true).Render(text) }

// spaced appends more after a blank row, when rows already holds something.
func spaced(rows []string, more ...string) []string {
	if len(rows) > 0 {
		rows = append(rows, "")
	}
	return append(rows, more...)
}
