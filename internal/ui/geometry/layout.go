package geometry

// Layout is the terminal box and what its chrome costs, as the body's height
// and a pane's pin read them.
type Layout struct {
	Width, Height int
	HeaderRows    int
	// FooterRows measures the painted footer; QuickFooterRows the one the
	// quick bar paints, read only while QuickBar says a bar is up. They are
	// read only for a body height: the quick footer sizes itself from
	// ChromeRows, so measuring them eagerly would recurse.
	FooterRows, QuickFooterRows func() int
	QuickBar                    bool
}

// ChromeRows is the rows above the body: the header band and the rule that
// closes it.
func (l Layout) ChromeRows() int {
	return l.HeaderRows + 1
}

func (l Layout) BodyHeight() int {
	return l.bodyRows(l.FooterRows())
}

// RestingBodyHeight is the same budget with the footer the screen actually
// paints: a growing footer crops nothing and resizes no running pane.
func (l Layout) RestingBodyHeight() int {
	if !l.QuickBar {
		return l.BodyHeight()
	}
	return l.bodyRows(l.QuickFooterRows())
}

func (l Layout) bodyRows(footer int) int {
	return max(l.Height-l.ChromeRows()-1-footer, 3)
}

// PaneSize is the tmux window size sessions are pinned to: the whole terminal
// body, so an attach opens at the width it already has.
func (l Layout) PaneSize() (int, int) {
	return max(l.Width, 1), l.RestingBodyHeight()
}

// BodyRange is the inclusive-start, exclusive-end row range of the sessions
// body.
func (l Layout) BodyRange() (start, end int) {
	start = l.ChromeRows()
	return start, start + l.BodyHeight()
}
