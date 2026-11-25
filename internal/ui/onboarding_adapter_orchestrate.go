package ui

import (
	uitheme "github.com/Doremi203/aghelm/internal/ui/theme"
	"time"

	"github.com/Doremi203/aghelm/internal/status"
	uionboarding "github.com/Doremi203/aghelm/internal/ui/onboarding"
	uirail "github.com/Doremi203/aghelm/internal/ui/rail"
	"github.com/charmbracelet/lipgloss"
)

// helper is one agent the lead opens: where it lands, what it says while it
// works and once it is done, and the beat of the coordinating scene it
// finishes on (0 keeps it working). A helper with a question waits from
// asks until answered.
type helper struct {
	name, group    string
	working, done  string
	finish         int
	question       string
	asks, answered int
}

var helpers = []helper{
	{name: "order schema", group: "checkout/api", working: "Drafting the order types", done: "Types are in api/order.ts", finish: 12},
	{name: "payment intents", group: "checkout/api", working: "Wiring the payment intent", done: "Intents create and confirm", finish: 36},
	{name: "cart page", group: "checkout/web", working: "Building the cart page", done: "The cart reads the new types", finish: 56},
	{name: "address form", group: "checkout/web", working: "Laying out the address form",
		question: "Which countries do we ship to?", asks: 16, answered: 28},
	{name: "pay button", group: "checkout/web", working: "Hooking up the pay button"},
	{name: "e2e flows", group: "checkout/tests", working: "Writing the happy path"},
	{name: "load test", group: "checkout/tests", working: "Ramping 200 carts a second"},
}

const (
	previewLead = "preview-lead"
	// helperArrival is the beats between two helpers opening.
	helperArrival = 7
	// helperStarting is how long a fresh helper shows as starting.
	helperStarting = 4
)

// orchestrateScene: one agent is asked to lead a rewrite, opens a team of
// agents sorted into groups, and coordinates them while the list follows.
func (m *Model) orchestrateScene(scene *previewScene, p uionboarding.Preview, width int) {
	now := time.Now()
	lead := uirail.Session{ID: previewLead, Name: "checkout lead", Tool: p.Agents[0], Status: status.Idle,
		Prompt: "keep checkout healthy", PaneLine: "Ready.", CreatedAt: now.Add(-time.Hour), LastStatusAt: now.Add(-time.Hour)}
	scene.sessions, scene.groups = []uirail.Session{lead}, nil
	scene.selected = sessionSelection(previewLead, "")
	target := lipgloss.NewStyle().Foreground(uitheme.ColorBright()).Render("checkout lead")
	ask := uionboarding.OrchestrateAsk
	sent := p.Scene > 0 || p.Frame >= len([]rune(ask))+10
	if !sent {
		scene.foot = m.previewBar(width, "answer", target, "", typedOut(ask, p.Frame-3))
		return
	}
	scene.foot = m.previewBar(width, "answer", target, "", "")
	scene.sessions[0].Status, scene.sessions[0].PaneLine = status.Working, "Coordinating 7 agents"
	switch p.Scene {
	case 0:
		scene.sessions[0].PaneLine = "Reading the checkout code"
	case 1:
		for i, h := range helpers {
			age := p.Frame - i*helperArrival
			if age < 0 {
				break
			}
			scene.sessions = append(scene.sessions, h.session(p.Agents, i, age < helperStarting, -1, now))
		}
	default:
		for i, h := range helpers {
			scene.sessions = append(scene.sessions, h.session(p.Agents, i, false, p.Frame, now))
		}
	}
}

// session is the helper's row: starting while it opens, then working,
// waiting on its question, or done, by the beat of the coordinating scene
// (-1 before that scene).
func (h helper) session(agents []string, index int, starting bool, beat int, now time.Time) uirail.Session {
	row := uirail.Session{ID: "preview-helper-" + h.name, Name: h.name, Group: h.group, Tool: agents[(index+1)%len(agents)],
		Status: status.Working, PaneLine: h.working, Prompt: h.working, CreatedAt: now, LastStatusAt: now}
	switch {
	case starting:
		row.Status, row.PaneLine = status.Starting, ""
	case h.finish > 0 && beat >= h.finish:
		row.Status, row.PaneLine = status.Finished, h.done
	case h.question != "" && beat >= h.asks && beat < h.answered:
		row.Status, row.PaneLine = status.Waiting, h.question
	}
	return row
}
