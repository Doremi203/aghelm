package sessioncmd

// Vocabulary is how one front spells the actions these errors send a caller
// after. Both fronts drive this layer, but an agent holds only one of them:
// a session whose CLI carries no MCP client has the subcommands and nothing
// else, so naming a tool at it points at something it cannot call.
type Vocabulary struct {
	ListSessions   string
	ListTerminals  string
	ListTasks      string
	ListGroups     string
	CreateGroup    string
	CreateTerminal string
	SpawnTool      string
	SpawnModel     string
	SpawnEffort    string
	SpawnProfile   string
	Revive         string
	Restore        string
	Send           string
	Read           string
	ArchiveSelf    string
	KillSelf       string
}

// MCPVocabulary spells the actions as the tools an MCP client sees.
func MCPVocabulary() Vocabulary {
	return Vocabulary{
		ListSessions:   "list_sessions",
		ListTerminals:  "list_terminals",
		ListTasks:      `task with action "list"`,
		ListGroups:     "list_groups",
		CreateGroup:    "create_group",
		CreateTerminal: "create_terminal",
		SpawnTool:      "create_session tool",
		SpawnModel:     "create_session model",
		SpawnEffort:    "create_session effort",
		SpawnProfile:   "create_session profile",
		Revive:         "revive_session",
		Restore:        "archive_session archived false",
		Send:           "send_session",
		Read:           "read_session",
		ArchiveSelf:    "archive_self",
		KillSelf:       "kill_self",
	}
}

// CLIVocabulary spells them as the subcommands an agent runs from its shell.
func CLIVocabulary() Vocabulary {
	return Vocabulary{
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
}
