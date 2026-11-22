package sessioncmd

import "testing"

func TestCLIVocabulary(t *testing.T) {
	want := Vocabulary{
		ListSessions:   "aghelm sessions",
		ListTerminals:  "aghelm terminal list",
		ListTasks:      "aghelm task list",
		ListGroups:     "aghelm groups",
		CreateGroup:    "aghelm create-group",
		CreateTerminal: "aghelm terminal create",
		SpawnTool:      "aghelm spawn --tool",
		SpawnModel:     "aghelm spawn --model",
		SpawnEffort:    "aghelm spawn --effort",
		SpawnProfile:   "aghelm spawn --profile",
		Revive:         "aghelm revive",
		Restore:        "aghelm archive --restore",
		Send:           "aghelm send",
		Read:           "aghelm read",
		ArchiveSelf:    "aghelm archive-self",
		KillSelf:       "aghelm kill-self",
	}
	if got := CLIVocabulary(); got != want {
		t.Fatalf("CLIVocabulary() = %+v, want %+v", got, want)
	}
}
