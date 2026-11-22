package frame

import (
	"strings"
	"testing"
)

func TestHeaderRowDropsVersionFirst(t *testing.T) {
	h := Header{
		Left:    "agent-manager",
		Version: " v0.40.1",
		Scope:   "repo",
		Rollup:  []string{"3 working", ""},
		Sep:     "   ",
		Gutter:  2,
	}
	for _, tc := range []struct {
		width int
		want  string
	}{
		{60, "agent-manager v0.40.1" + strings.Repeat(" ", 21) + "repo   3 working  "},
		{34, "agent-manager" + strings.Repeat(" ", 3) + "repo   3 working  "},
		{24, "agent-manager" + strings.Repeat(" ", 5) + "repo  "},
	} {
		h.Width = tc.width
		if got := h.Row(); got != tc.want {
			t.Errorf("width %d:\n got %q\nwant %q", tc.width, got, tc.want)
		}
	}
}
