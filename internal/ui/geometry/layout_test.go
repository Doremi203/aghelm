package geometry

import "testing"

func testLayout() Layout {
	return Layout{
		Width: 100, Height: 40,
		HeaderRows: 2,
		FooterRows: func() int { return 1 },
	}
}

// The body's height is the terminal minus the chrome the frame paints.
func TestLayoutBodyHeight(t *testing.T) {
	l := testLayout()
	if got := l.ChromeRows(); got != 3 {
		t.Fatalf("ChromeRows = %d, want the header band and its rule", got)
	}
	if got := l.BodyHeight(); got != 40-3-1-1 {
		t.Fatalf("BodyHeight = %d", got)
	}
	if start, end := l.BodyRange(); start != 3 || end != 3+l.BodyHeight() {
		t.Fatalf("BodyRange = %d..%d", start, end)
	}
}

// The quick bar paints a taller footer, so the resting body gives way.
func TestLayoutQuickBarShortensTheRestingBody(t *testing.T) {
	l := testLayout()
	plain := l.RestingBodyHeight()
	l.QuickBar = true
	l.QuickFooterRows = func() int { return 3 }
	if got := l.RestingBodyHeight(); got >= plain {
		t.Fatalf("the quick bar's footer did not shorten the body: %d >= %d", got, plain)
	}
}

// Sessions are pinned to the whole body.
func TestLayoutPaneSizeIsTheWholeBody(t *testing.T) {
	l := testLayout()
	if w, h := l.PaneSize(); w != 100 || h != l.RestingBodyHeight() {
		t.Fatalf("PaneSize = %d,%d, want the whole body", w, h)
	}
}
