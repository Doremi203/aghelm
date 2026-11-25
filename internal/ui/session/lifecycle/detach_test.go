package lifecycle

import (
	"errors"
	"testing"
	"time"

	"github.com/Doremi203/aghelm/internal/status"
	"github.com/Doremi203/aghelm/internal/store"
	"github.com/Doremi203/aghelm/internal/tmux"
	tea "github.com/charmbracelet/bubbletea"
)

// The acknowledgement clears the unread mark the finished status left.
func TestAckRequestClearsTheUnreadMark(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir() + "/state.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	sess := store.Session{ID: "s1", Name: "one", Tool: "claude", Cwd: t.TempDir(), CreatedAt: time.Now(), Status: status.Idle}
	if err := st.CreateSession(sess); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateStatus(sess.ID, status.Finished); err != nil {
		t.Fatal(err)
	}
	if _, err := (AckRequest{SessionID: sess.ID}).Run(Services{Store: st}); err != nil {
		t.Fatalf("ack: %v", err)
	}
	acked, err := st.Get(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if acked.Status != status.Idle || !acked.Acked {
		t.Fatalf("after the ack, status = %q acked = %v", acked.Status, acked.Acked)
	}
}

type detachHost struct {
	errs, warns []string
	refreshed   bool
	quitting    bool
	taken       bool
	gone        bool
	opened      string
}

func (h *detachHost) ReportErr(text string)        { h.errs = append(h.errs, text) }
func (h *detachHost) Warn(text string)             { h.warns = append(h.warns, text) }
func (h *detachHost) Refresh()                     { h.refreshed = true }
func (h *detachHost) Quitting() bool               { return h.quitting }
func (h *detachHost) TakenSinceDetach() bool       { return h.taken }
func (h *detachHost) Name(sessionID string) string { return "agent " + sessionID }
func (h *detachHost) Select(string) bool           { return !h.gone }
func (h *detachHost) Review(string) tea.Cmd        { h.opened = "review"; return nil }
func (h *detachHost) Editor(string) tea.Cmd        { h.opened = "editor"; return nil }

// The request a session left on its way out opens the view it named, and
// any reason not to is reported instead of opening something stale.
func TestApplyDetachOpensWhatTheSessionAskedFor(t *testing.T) {
	t.Parallel()
	request := DetachRequest{SessionID: "s1"}
	for _, tc := range []struct {
		name    string
		host    detachHost
		result  Detach
		err     error
		opened  string
		errs    int
		warns   int
		refresh bool
	}{
		{name: "review", result: Detach{Request: tmux.RequestReview}, opened: "review"},
		{name: "editor", result: Detach{Request: tmux.RequestEditor}, opened: "editor"},
		{name: "no request", refresh: true},
		{name: "read failed", err: errors.New("no server"), errs: 1},
		{name: "clear failed", result: Detach{Request: tmux.RequestReview, ClearErr: errors.New("busy")}, errs: 1, refresh: true},
		{name: "quitting", host: detachHost{quitting: true}, result: Detach{Request: tmux.RequestReview}},
		{name: "screen taken", host: detachHost{taken: true}, result: Detach{Request: tmux.RequestReview}, warns: 1, refresh: true},
		{name: "row gone", host: detachHost{gone: true}, result: Detach{Request: tmux.RequestEditor}, errs: 1, refresh: true},
	} {
		host := tc.host
		ApplyDetach(&host, request, tc.result, tc.err)
		if host.opened != tc.opened || len(host.errs) != tc.errs || len(host.warns) != tc.warns || host.refreshed != tc.refresh {
			t.Errorf("%s: opened %q errs %v warns %v refreshed %v", tc.name, host.opened, host.errs, host.warns, host.refreshed)
		}
	}
}
