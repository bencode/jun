package app

import (
	"fmt"
	"slices"
	"testing"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/egoist/mygo/ui"
)

func peekLabel(id terminal.ID) string { return fmt.Sprintf("Peek terminal %d", id) }

// onlyWaiting leaves waiting only the agents given.
func onlyWaiting(a *app, waiting ...*terminal.Session) {
	for id, info := range a.agents {
		info.Status = agent.Idle
		a.agents[id] = info
	}
	for _, s := range waiting {
		a.agents[s.ID] = agentInfo{agent.Claude, agent.Waiting}
	}
}

func shownSession(t *testing.T, a *app, sessions []*terminal.Session) *terminal.Session {
	t.Helper()
	for _, s := range sessions {
		if a.terminalOnScreen(s.ID) {
			return s
		}
	}
	t.Fatal("no agent's terminal shows")
	return nil
}

func TestPeekKeepsTheTerminalsGrid(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	s := shownSession(t, a, sessions)
	onlyWaiting(a, s)
	cols, rows := s.Terminal.Size()
	jun := a.state.CurrentProject()
	if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	tt.Key(ui.Super, ui.KeyJ)
	tt.Frame()
	tt.Frame()
	if a.peek.id != s.ID || !tt.Focused(peekLabel(s.ID)) {
		t.Fatalf("⌘J did not peek and focus the waiting agent: peek %d", a.peek.id)
	}
	if c, r := s.Terminal.Size(); c != cols || r != rows {
		t.Fatalf("peek resized the agent from %dx%d to %dx%d", cols, rows, c, r)
	}
	tt.Key(ui.Super, ui.KeyEnter)
	tt.Frame()
	tt.Frame()
	if a.peek.id != 0 || a.state.CurrentProject() != jun || !tt.Focused(fmt.Sprintf("Terminal content %d", s.ID)) {
		t.Fatal("⌘↩ did not close the peek and show the agent's tab")
	}
	if c, r := s.Terminal.Size(); c != cols || r != rows {
		t.Fatalf("back in its tab the agent is %dx%d, was %dx%d", c, r, cols, rows)
	}
}

func TestPeekNextGoesThroughWaitingAgents(t *testing.T) {
	a, tt, _ := agentsApp(t)
	addAgents(t, a, agent.Waiting, agent.Waiting)
	all := a.terminals.All()
	first, second := all[len(all)-2], all[len(all)-1]
	onlyWaiting(a, first, second)
	tt.Frame()
	for _, want := range []*terminal.Session{first, second} {
		tt.Key(ui.Super, ui.KeyJ)
		tt.Frame()
		if a.peek.id != want.ID {
			t.Fatalf("⌘J peeked %d, want %d", a.peek.id, want.ID)
		}
	}
	a.agents[first.ID] = agentInfo{agent.Claude, agent.Working}
	tt.Key(ui.Super, ui.KeyJ)
	tt.Frame()
	if a.peek.id != 0 {
		t.Fatalf("⌘J with no other agent waiting left %d peeked", a.peek.id)
	}
}

func TestPeekLeavesEscapeToTheAgentAndCommandWClosesIt(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	s := shownSession(t, a, sessions)
	click(t, tt, fmt.Sprintf("Terminal content %d", s.ID))
	addAgents(t, a, agent.Waiting)
	all := a.terminals.All()
	hidden := all[len(all)-1]
	onlyWaiting(a, hidden)
	region := a.targetRegion()
	views := len(region.Views)
	tt.Frame()
	tt.Key(ui.Super, ui.KeyJ)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Esc → agent") || !tt.HasText("Close ⌘J") {
		t.Fatal("the drawer does not say where Escape goes, or that ⌘J closes it on the last waiting agent")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.peek.id != hidden.ID || !tt.Focused(peekLabel(hidden.ID)) || !tt.HasText("Esc went to the agent · ⌘W closes") {
		t.Fatal("Escape closed the peek, took the focus from its terminal, or showed no way out")
	}
	tt.Key(ui.Super, ui.KeyW)
	tt.Frame()
	tt.Frame()
	if a.peek.id != 0 || len(region.Views) != views || !tt.Focused(fmt.Sprintf("Terminal content %d", s.ID)) {
		t.Fatalf("⌘W did not close only the peek, giving the focus back: peek %d views %d/%d focused %v", a.peek.id, len(region.Views), views, tt.Focused(fmt.Sprintf("Terminal content %d", s.ID)))
	}
}

func TestPeekOfAShownTerminalFocusesItsTab(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	s := shownSession(t, a, sessions)
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	click(t, tt, a.agentLabel(s))
	tt.Frame()
	if a.peek.id != 0 || !tt.Focused(fmt.Sprintf("Terminal content %d", s.ID)) {
		t.Fatal("clicking an agent whose tab shows did not focus the tab")
	}
}

func TestClickingThePeekedCardClosesIt(t *testing.T) {
	a, tt, _ := agentsApp(t)
	addAgents(t, a, agent.Waiting)
	all := a.terminals.All()
	hidden := all[len(all)-1]
	tt.Frame()
	click(t, tt, a.agentLabel(hidden))
	tt.Frame()
	if a.peek.id != hidden.ID {
		t.Fatal("clicking the card did not peek its agent")
	}
	click(t, tt, a.agentLabel(hidden))
	tt.Frame()
	if a.peek.id != 0 {
		t.Fatal("clicking the peeked card again did not close the drawer")
	}
}

func TestNextWaiting(t *testing.T) {
	order := []terminal.ID{1, 2, 3, 4}
	waiting := func(ids ...terminal.ID) func(terminal.ID) bool {
		return func(id terminal.ID) bool { return slices.Contains(ids, id) }
	}
	for _, c := range []struct {
		name    string
		waiting []terminal.ID
		after   terminal.ID
		want    terminal.ID
	}{
		{"from none, the first", []terminal.ID{2, 4}, 0, 2},
		{"after one, the next", []terminal.ID{2, 4}, 2, 4},
		{"wraps around", []terminal.ID{1, 3}, 3, 1},
		{"never after itself", []terminal.ID{3}, 3, 0},
		{"none waiting", nil, 1, 0},
	} {
		if got := nextWaiting(order, waiting(c.waiting...), c.after); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}
