package notices

import (
	"slices"
	"testing"
)

func TestActiveListsTheConfigImportWarning(t *testing.T) {
	src := Sources{Store: true, ConfigImportError: "bad key"}
	notices := NewPanel(map[string]bool{}, map[string]bool{}).Active(src)
	if got := ids(notices); !slices.Equal(got, []string{ConfigNotImported}) {
		t.Fatalf("ids = %v, want the config warning alone", got)
	}
	n := notices[0]
	if n.Kind != KindHeadsUp || n.Subtitle != "Settings could not import it" || !slices.Contains(n.Body, "bad key") {
		t.Fatalf("config warning = %+v", n)
	}
	if !n.Unread {
		t.Fatal("an unseen warning reads as read")
	}
}

func TestActiveWithoutAWarningListsNothing(t *testing.T) {
	if notices := NewPanel(map[string]bool{}, map[string]bool{}).Active(Sources{Store: true}); len(notices) != 0 {
		t.Fatalf("notices = %v", ids(notices))
	}
}
