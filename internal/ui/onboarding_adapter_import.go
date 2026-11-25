package ui

import (
	"fmt"
	uilaunch "github.com/Doremi203/aghelm/internal/ui/spawn/launch"
	"time"

	"github.com/Doremi203/aghelm/internal/status"
	uionboarding "github.com/Doremi203/aghelm/internal/ui/onboarding"
	uirail "github.com/Doremi203/aghelm/internal/ui/rail"
	tea "github.com/charmbracelet/bubbletea"
)

const importSessionName = "import-past-chats"

// startSetupImport opens the agent that brings the picked CLIs' saved
// conversations in, at the root, the way a spawn from the list would.
func (m *Model) startSetupImport(request uionboarding.ImportRequest) tea.Cmd {
	defaults := m.spawnDefaults("")
	m.clearErr()
	m.dispatchSpawn(uilaunch.Request{
		Kind:         uilaunch.Import,
		ToolName:     request.Runner,
		Name:         importSessionName,
		Prompt:       uionboarding.ImportPrompt(request),
		Pane:         defaults.Pane,
		DirFallbacks: defaults.Fallbacks,
	})
	return m.nextEffectCmd()
}

// pastChats are the conversations the import scene brings in.
var pastChats = []string{"auth token refresh", "flaky e2e retry", "billing webhook", "readme rewrite"}

const (
	previewImporter = "preview-importer"
	// chatArrival is the beats between two chats coming in, after the
	// first beat the scene finds them on.
	chatArrival = 10
	chatsFound  = 12
)

// importScene: the import agent finds the picked CLIs' saved chats and the
// list fills with them one after another.
func (m *Model) importScene(scene *previewScene, p uionboarding.Preview) {
	from, runner := p.Import.From, p.Import.Runner
	if len(from) == 0 {
		from = p.Agents
	}
	if runner == "" {
		runner = p.Agents[0]
	}
	now := time.Now()
	arrived := 0
	if p.Frame >= chatsFound {
		arrived = min(1+(p.Frame-chatsFound)/chatArrival, len(pastChats))
	}
	importer := uirail.Session{ID: previewImporter, Name: "import past chats", Tool: runner, Status: status.Working,
		PaneLine: "Looking through saved chats", CreatedAt: now, LastStatusAt: now}
	for i, name := range pastChats[:arrived] {
		scene.sessions = append(scene.sessions, uirail.Session{ID: "preview-chat-" + name, Name: name, Tool: from[i%len(from)],
			Status: status.Idle, CreatedAt: now, LastStatusAt: now})
	}
	if arrived == len(pastChats) && p.Frame >= chatsFound+len(pastChats)*chatArrival {
		importer.Status, importer.PaneLine = status.Finished, fmt.Sprintf("Brought in %d chats.", len(pastChats))
	}
	scene.sessions = append([]uirail.Session{importer}, scene.sessions...)
	scene.selected = sessionSelection(previewImporter, "")
}
