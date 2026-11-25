package workspace

import (
	"fmt"
	"strings"

	uiframe "github.com/Doremi203/aghelm/internal/ui/frame"
	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
)

// railInset is the blank run the foot keeps left of its readings, matching
// the Rail's gutter.
const railInset = uiframe.RailGutter - 1

func tempReadings(r Readings) string {
	styled := make([]string, 0, 3)
	for _, reading := range Temperatures(r) {
		styled = append(styled, uitheme.ValueStyle().Render(reading))
	}
	return strings.Join(styled, uitheme.SubtleStyle().Render("  "))
}

// FootLine is the machine readings condensed to one line, for the full
// screen layout whose rail has a single row to spare.
func FootLine(r Readings, net Net, width int, mode presentation.GlyphMode) []string {
	reading := func(label, value string, ok bool) string {
		if !ok {
			return uitheme.LabelStyle().Render(label+" ") + uitheme.SubtleStyle().Render("n/a")
		}
		return uitheme.LabelStyle().Render(label+" ") + uitheme.ValueStyle().Render(value)
	}
	parts := []string{
		reading("cpu", fmt.Sprintf("%.0f%%", r.CPUPercent), r.CPUOK),
		reading("mem", fmt.Sprintf("%.0f%% %s/%s", r.MemPercent, uiframe.HumanBytes(r.MemUsed), uiframe.HumanBytes(r.MemTotal)), r.MemOK),
	}
	if r.SwapOK && r.SwapTotal > 0 {
		parts = append(parts, reading("swap", fmt.Sprintf("%.0f%% %s/%s", r.SwapPercent, uiframe.HumanBytes(r.SwapUsed), uiframe.HumanBytes(r.SwapTotal)), true))
	}
	parts = append(parts, reading("disk", fmt.Sprintf("%.0f%% %s available", r.DiskPercent, uiframe.DiskBytes(r.DiskAvailable)), r.DiskOK))
	if r.BatteryOK {
		value := fmt.Sprintf("%.0f%%", r.BatteryPercent)
		if r.BatteryCharging {
			value += " charging"
		}
		parts = append(parts, reading("batt", value, true))
	}
	if temps := tempReadings(r); temps != "" {
		parts = append(parts, uitheme.LabelStyle().Render("temp ")+temps)
	}
	if net.Rates {
		nav := presentation.NavigationFor(mode)
		parts = append(parts, reading("net", nav.DownGlyph+" "+uiframe.HumanBytes(net.Down)+"/s "+nav.UpGlyph+" "+uiframe.HumanBytes(net.Up)+"/s", true))
	}
	line := strings.Repeat(" ", railInset) + strings.Join(parts, "  ")
	return []string{ansi.Truncate(line, max(width-railInset, 0), "…")}
}
