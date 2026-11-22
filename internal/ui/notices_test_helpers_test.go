package ui

import (
	uinotices "github.com/Doremi203/aghelm/internal/ui/notices"
)

// noticesSurfaceOf is the messages modal on the stack, which a test reads the
// cursor and scroll from.
func noticesSurfaceOf(m *Model) *uinotices.Surface {
	s, ok := surfaceOf[*uinotices.Surface](m)
	if !ok {
		panic("the messages modal is not open")
	}
	return s
}

// noticeTip is the notice withNotice lists: the refused config import, the
// one notice the binary builds, which has a link to open and can be
// dismissed.
const noticeTip = uinotices.ConfigNotImported

// withNotice lists noticeTip, its reason reading reason.
func withNotice(m *Model, reason string) {
	m.notices.ImportError = reason
}
