package connections

import (
	uirail "github.com/Doremi203/aghelm/internal/ui/rail"
)

func (c *Connections) Selected(selection uirail.Selection) (Row, bool) {
	if !selection.Remote() {
		return Row{}, false
	}
	kind := SessionRow
	switch selection.Kind {
	case uirail.ConnectionRow:
		kind = ConnectionRow
	case uirail.GroupRow:
		kind = GroupRow
	}
	return c.Row(Target{Kind: kind, Host: selection.Host, Group: selection.Group, ID: selection.SessionID})
}

// AimLost refuses an enter whose target a refresh moved, when either end of
// the aim is on a connection: the row the bar was aimed at is gone, or the
// cursor slid to another row without the user moving it.
func (c *Connections) AimLost(aim, selection uirail.Selection) (string, bool) {
	if !aim.Remote() && !selection.Remote() {
		return "", false
	}
	if _, ok := c.Selected(aim); aim.Remote() && !ok {
		return "the agent this prompt was aimed at is gone from " + aim.Host + "; nothing was sent", true
	}
	if !SameRow(aim, selection) {
		return "the selection moved since this prompt was aimed; nothing was sent", true
	}
	return "", false
}

// SameRow compares rows by identity; a session that changed group is still
// the same session.
func SameRow(a, b uirail.Selection) bool {
	if a.Kind == uirail.SessionRow && b.Kind == uirail.SessionRow {
		return a.Host == b.Host && a.SessionID == b.SessionID
	}
	return a == b
}
