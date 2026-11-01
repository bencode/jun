package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// agentsApp marks the test app's terminals as agents, as the poller would.
func agentsApp(t *testing.T) (*app, *ui.Tester, []*terminal.Session) {
	t.Helper()
	a, tt := testApp(t)
	t.Cleanup(a.closePreviews)
	sessions := a.terminals.All()
	if len(sessions) != 3 {
		t.Fatalf("want 3 terminals, got %d", len(sessions))
	}
	a.agents[sessions[0].ID] = agentInfo{agent.Claude, agent.Idle}
	a.agents[sessions[1].ID] = agentInfo{agent.Codex, agent.Working}
	a.agents[sessions[2].ID] = agentInfo{agent.Pi, agent.Waiting}
	tt.Frame()
	tt.Frame()
	return a, tt, sessions
}

func row(t *testing.T, tt *ui.Tester, a *app, s *terminal.Session) ui.Rect {
	t.Helper()
	r, ok := tt.Find(a.agentLabel(s))
	if !ok {
		t.Fatalf("agent %d not shown", s.ID)
	}
	return r
}

// addAgents starts terminals running agents in the given statuses.
func addAgents(t *testing.T, a *app, statuses ...agent.Status) {
	t.Helper()
	for _, status := range statuses {
		s, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
		if err != nil {
			t.Fatal(err)
		}
		a.agents[s.ID] = agentInfo{agent.Claude, status}
	}
}

func resize(tt *ui.Tester, width, height int) {
	tt.SetSize(width, height)
	tt.Frame()
	tt.Frame()
}

func TestAgentsLayoutFitsRoom(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	if len(a.previews) != 3 || row(t, tt, a, sessions[2]).H < cardMinHeight {
		t.Fatal("three agents in Global did not show as cards")
	}
	addAgents(t, a, agent.Working, agent.Idle, agent.Idle)
	resize(tt, 1440, 900)
	idle, working, waiting := row(t, tt, a, sessions[0]), row(t, tt, a, sessions[1]), row(t, tt, a, sessions[2])
	if !(waiting.Y < working.Y && working.Y < idle.Y) || waiting.H != rowHeight {
		t.Fatal("six agents did not list in rows, waiting first")
	}
	for _, count := range []string{"All 6", "Waiting 1", "Working 2", "Idle 3"} {
		if !tt.HasText(count) {
			t.Fatalf("missing count %q", count)
		}
	}
	resize(tt, 1440, 420)
	if row(t, tt, a, sessions[0]).H != compactRowHeight {
		t.Fatal("agents did not list in compact rows when rows would not fit")
	}
	resize(tt, 1440, 330)
	if _, ok := tt.Find(a.agentLabel(sessions[0])); ok {
		t.Fatal("idle agents were not folded when compact rows would not fit")
	}
	click(t, tt, "Idle 3 ▸")
	row(t, tt, a, sessions[0])
}

func TestAgentsLayoutChoice(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	click(t, tt, "Agents layout")
	if err := tt.ChooseMenuItem("List"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if row(t, tt, a, sessions[0]).H != rowHeight || a.state.Global.Views[0].AgentsLayout != layout.AgentsList {
		t.Fatal("choosing List did not list the agents, or was not kept in the layout")
	}
}

func TestAgentClickRevealsTabAcrossProjects(t *testing.T) {
	a, tt, _ := agentsApp(t)
	jun := a.state.CurrentProject()
	review := jun.Perspectives[1]
	view := review.Bottom.Views[0]
	review.BottomVisible = false
	if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	click(t, tt, a.agentLabel(a.sessionFor(view.ID)))
	if a.state.CurrentProject() != jun || a.state.CurrentPerspective() != review || !review.BottomVisible ||
		review.Bottom.Active != view.ID || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("click did not reveal and focus the agent's tab")
	}
}

func TestAgentClickReopensClosedTabInItsProject(t *testing.T) {
	a, tt, _ := agentsApp(t)
	jun := a.state.CurrentProject()
	hidden, err := a.terminals.Create("Jun", jun.CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.agents[hidden.ID] = agentInfo{agent.Claude, agent.Waiting}
	if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	click(t, tt, a.agentLabel(hidden))
	column := a.state.CurrentPerspective().Columns[0]
	view := column.Views[len(column.Views)-1]
	if a.state.CurrentProject() != jun || column.Active != view.ID || a.bindings[view.ID] != hidden.ID ||
		view.ProjectName != "Jun" || len(a.state.Global.Views) != 1 || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("click did not reopen the agent's session in its project's column")
	}
}

func TestPreviewSkipsHalfDrawnSynchronizedUpdates(t *testing.T) {
	source, err := ghostty.New(ghostty.Options{Conn: &discardConn{done: make(chan struct{})}, ResizeByConn: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { source.Close() })
	source.Resize(20, 4)
	source.Feed([]byte("before"))
	p, err := newPreview(func() {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.term.Close() })
	p.refresh(source, 1)
	source.Feed([]byte("\x1b[?2026h\x1b[2K\r"))
	p.refresh(source, 2)
	if !strings.Contains(p.term.Text(), "before") {
		t.Fatalf("preview showed a half-drawn screen: %q", p.term.Text())
	}
	source.Feed([]byte("after\x1b[?2026l"))
	p.refresh(source, 2)
	if text := p.term.Text(); !strings.Contains(text, "after") || strings.Contains(text, "before") {
		t.Fatalf("preview missed the finished update: %q", text)
	}
}

func TestPreviewShowsTheBottomOfTheScreenItsViewFits(t *testing.T) {
	source, err := ghostty.New(ghostty.Options{Conn: &discardConn{done: make(chan struct{})}, ResizeByConn: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { source.Close() })
	source.Resize(30, 10)
	source.Feed([]byte("\r\n\r\n\r\n\r\nolder\r\n> box\r\n  status"))
	p, err := newPreview(func() {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.term.Close() })
	p.refresh(source, 1)
	p.conn.Resize(80, 2)
	if cols, rows := p.term.Size(); cols != 30 || rows != 2 || p.term.Text() != "> box\n  status" {
		t.Fatalf("preview %dx%d shows %q", cols, rows, p.term.Text())
	}
}

func TestRailShowsWaitingAgents(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	for _, expanded := range []bool{false, true} {
		a.state.RailExpanded = expanded
		tt.Frame()
		if _, ok := tt.Find("1 waiting"); !ok {
			t.Fatalf("rail (expanded: %v) did not mark the waiting agent", expanded)
		}
	}
	a.agents[sessions[2].ID] = agentInfo{agent.Pi, agent.Working}
	tt.Frame()
	if _, ok := tt.Find("1 waiting"); ok {
		t.Fatal("rail still marks an agent that stopped waiting")
	}
}

func TestAgentsHighlightLastFocusedTerminal(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	var shown []*terminal.Session
	for _, s := range sessions {
		if _, ok := tt.Find(fmt.Sprintf("Terminal content %d", s.ID)); ok {
			shown = append(shown, s)
		}
	}
	if len(shown) < 2 {
		t.Fatalf("want two agents' terminals shown, got %d", len(shown))
	}
	current := func(s *terminal.Session) bool { return strings.HasSuffix(a.agentLabel(s), ", current") }
	for _, s := range shown[:2] {
		click(t, tt, fmt.Sprintf("Terminal content %d", s.ID))
		tt.Frame()
		row(t, tt, a, s)
		if !current(s) {
			t.Fatalf("agent %d was not current after its terminal was focused", s.ID)
		}
	}
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	if !current(shown[1]) || current(shown[0]) {
		t.Fatal("the current agent changed when the focus left the terminals")
	}
}
