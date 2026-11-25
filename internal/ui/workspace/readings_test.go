package workspace

import (
	"strings"
	"testing"

	"github.com/Doremi203/aghelm/internal/ui/presentation"
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func forceANSI256(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// sgrOf returns the color sequence a style emits under the live theme, so a
// contrast assertion tracks the palette instead of a hardcoded index.
func sgrOf(rendered string) string {
	_, seq, found := strings.Cut(rendered, "\x1b[")
	if !found {
		return ""
	}
	code, _, _ := strings.Cut(seq, "m")
	return code
}

func lastSGR(s string) string {
	idx := strings.LastIndex(s, "\x1b[")
	if idx < 0 {
		return ""
	}
	code, _, _ := strings.Cut(s[idx+2:], "m")
	return code
}

// The separator carries its own reset, so a reading cannot inherit color.
func TestTemperatureReadingsEachKeepTheirColor(t *testing.T) {
	forceANSI256(t)

	row := tempReadings(Readings{CPUTempOK: true, CPUTemp: 61, GPUTempOK: true, GPUTemp: 55})
	want := sgrOf(uitheme.ValueStyle().Render("x"))
	for _, reading := range []string{"cpu 61°C", "gpu 55°C"} {
		before, _, found := strings.Cut(row, reading)
		if !found {
			t.Fatalf("row %q is missing %q", row, reading)
		}
		if got := lastSGR(before); got != want {
			t.Fatalf("%q renders under %q, want %q", reading, got, want)
		}
	}
}

// Full screen condenses the machine readings to one line.
func TestFootLineCondensesEveryReading(t *testing.T) {
	net := Net{Rates: true, Down: 1000, Up: 500}
	line := ansi.Strip(strings.Join(FootLine(Readings{
		CPUPercent: 22, CPUOK: true,
		MemPercent: 75, MemOK: true,
		DiskOK: true, DiskPercent: 66, DiskAvailable: 167_950_000_000,
	}, net, 119, presentation.DefaultGlyphMode), "\n"))
	for _, want := range []string{"cpu 22%", "mem 75%", "disk 66% 167.9GB available", "net"} {
		if !strings.Contains(line, want) {
			t.Errorf("condensed foot line misses %q:\n%s", want, line)
		}
	}
}

func TestFootLineFitsItsWidth(t *testing.T) {
	line := FootLine(Readings{
		CPUPercent: 22, CPUOK: true, MemPercent: 75, MemOK: true,
		DiskOK: true, DiskPercent: 66, DiskAvailable: 167_950_000_000,
	}, Net{Rates: true, Down: 1_000_000, Up: 500_000}, 40, presentation.DefaultGlyphMode)
	if got := ansi.StringWidth(line[0]); got > 39 {
		t.Fatalf("foot line %d cells wide: %q", got, ansi.Strip(line[0]))
	}
}
