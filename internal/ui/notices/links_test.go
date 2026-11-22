package notices

import (
	"strings"
	"testing"
)

func TestBugReportURLPrefillsVersion(t *testing.T) {
	got := BugReportURL("v0.2.0")
	if !strings.HasPrefix(got, "https://github.com/Doremi203/aghelm/issues/new") {
		t.Fatalf("bug report should open the new-issue page, got %q", got)
	}
	if !strings.Contains(got, "version=v0.2.0") {
		t.Fatalf("issue URL should carry the version, got %q", got)
	}
}

func TestFeatureRequestURLOpensABlankIssue(t *testing.T) {
	if got := FeatureRequestURL(); got != RepoURL+"/issues/new" {
		t.Fatalf("feature request should open the new-issue page, got %q", got)
	}
}

func TestSupportRequestURLPrefillsTheCLIFields(t *testing.T) {
	got := SupportRequestURL()
	if !strings.HasPrefix(got, RepoURL+"/issues/new?body=") {
		t.Fatalf("support request should open a prefilled issue, got %q", got)
	}
	for _, want := range []string{"CLI+name", "How+to+launch+it", "Config+and+tool+support"} {
		if !strings.Contains(got, want) {
			t.Fatalf("support request is missing %q: %q", want, got)
		}
	}
}

func TestQueuedFindsARunningOrWaitingDismissal(t *testing.T) {
	requests := []DismissRequest{{ID: "welcome"}}
	if !Queued(requests, "welcome") {
		t.Fatal("a queued dismissal should be found")
	}
	if Queued(requests, "feed-new") {
		t.Fatal("an unqueued id must not report queued")
	}
}

func TestFailedDismissalReopensOnlyBeforeTheUserMovesOn(t *testing.T) {
	request := DismissRequest{ID: "welcome", ForegroundGen: 7, Modal: true}
	stale := FailedDismissal(request, errString("disk full"), false, 8, false, true)
	if stale.Reopen != "" {
		t.Fatalf("a key pressed since accepting must not reopen the card: %+v", stale)
	}
	if stale.Restore != "welcome" || stale.Err == nil {
		t.Fatalf("the dismissal should still restore and report: %+v", stale)
	}
	fresh := FailedDismissal(request, errString("disk full"), false, 7, false, true)
	if fresh.Reopen != "welcome" {
		t.Fatalf("a clear list before any key should reopen the card: %+v", fresh)
	}
	quitting := FailedDismissal(request, errString("disk full"), true, 7, false, true)
	if quitting.Reopen != "" {
		t.Fatalf("a quitting manager must not reopen a card: %+v", quitting)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

// A refresh that adds nothing keeps the notice held while the list was busy;
// clearing it lost the popup the user was promised on return.
