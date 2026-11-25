package frame

import (
	"github.com/Doremi203/aghelm/internal/ui/presentation"
)

// Footer is the footer's height rule: the base screen's tiers and the
// measurement a transient tier is painted against.
type Footer struct {
	Mode        presentation.GlyphMode
	List        []Section
	Measurement Measurement
	Width       int
}

func (f Footer) Layout() (LegendMode, int) { return f.Measurement.Mode, f.Measurement.Rows }

func (f Footer) ListLegend() Legend {
	mode, rows := f.Layout()
	badge, quiet := TitleStyles()
	if mode == Titled {
		bar, _ := PaintTitledLegend(f.List, f.Width, LiveCap(f.Mode), badge, quiet, Separator())
		return bar
	}
	bar := PaintUntitledLegend(f.List, f.Width, LiveCap(f.Mode), mode, rows)
	bar.Text = presentation.PadToHeight(bar.Text, rows)
	return bar
}

// Transient paints one tier in the rows it needs, handing the rest to the
// body.
func (f Footer) Transient(tier Section) Legend {
	mode, rows := f.Layout()
	badge, quiet := TitleStyles()
	if mode == Titled {
		painted, _ := PaintTitledLegend([]Section{tier}, f.Width, LiveCap(f.Mode), badge, quiet, Separator())
		return painted
	}
	return PaintNarrowLegend([]Section{tier}, f.Width, LiveCap(f.Mode), mode, rows)
}
