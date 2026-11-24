package app

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/egoist/mygo/ui"
)

// agentsApp marks the test app's terminals as agents, as the poller would.
func agentsApp(t *testing.T) (*app, *ui.Tester, []*terminal.Session) {
	t.Helper()
	a, tt := testApp(t)
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

func TestAgentsListFitsRoom(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	a.state.CenterPercent = 76
	for _, k := range []layout.ViewKind{layout.ViewTasks, layout.ViewMessages} {
		a.toggleGlobalView(a.globalOf(k).ID) // the room is all Agents'
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

func TestAgentCommandClickRevealsTabAcrossProjects(t *testing.T) {
	a, tt, _ := agentsApp(t)
	jun := a.state.CurrentProject()
	review := jun.Perspectives[1]
	view := review.Bottom.Views[0]
	review.BottomVisible = false
	if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.ClickWith(ui.Super, a.agentLabel(a.sessionFor(view.ID))); err != nil {
		t.Fatal(err)
	}
	if a.state.CurrentProject() != jun || a.state.CurrentPerspective() != review || !review.BottomVisible ||
		review.Bottom.Active != view.ID || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("click did not reveal and focus the agent's tab")
	}
}

func TestAgentCommandClickReopensClosedTabInItsProject(t *testing.T) {
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
	if err := tt.ClickWith(ui.Super, a.agentLabel(hidden)); err != nil {
		t.Fatal(err)
	}
	column := a.state.CurrentPerspective().Columns[0]
	view := column.Views[len(column.Views)-1]
	if a.state.CurrentProject() != jun || column.Active != view.ID || a.bindings[view.ID] != hidden.ID ||
		view.ProjectName != "Jun" || len(a.state.Global.Views) != 3 || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("click did not reopen the agent's session in its project's column")
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

func TestAgentsGroupUnderTheirLead(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	idle, lead, waiting := sessions[0], sessions[1], sessions[2]
	addAgents(t, a, agent.Idle, agent.Waiting, agent.Idle, agent.Waiting)
	all := a.terminals.All()
	m1, m2, nested, orphan := all[3], all[4], all[5], all[6]
	shell, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.leads[m1.ID], a.leads[m2.ID], a.leads[nested.ID], a.leads[orphan.ID] = lead.ID, lead.ID, m1.ID, shell.ID
	resize(tt, 1440, 900)

	// The lead's group comes with the waiting agents, as one of its own
	// waits; the lead first, then the rest of the group by id.
	want := []terminal.ID{lead.ID, m1.ID, m2.ID, nested.ID, waiting.ID, orphan.ID, idle.ID}
	if got := a.agentOrder(); !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
	for _, heading := range []string{"Waiting 6", "Idle 1"} {
		if !tt.HasText(heading) {
			t.Fatalf("missing heading %q", heading)
		}
	}
	// The group's line marks its lead; an agent led by another, or by no
	// running agent, is marked.
	mark := func(id terminal.ID) bool { return tt.HasText(fmt.Sprintf("↳ #%d", id)) }
	at := func(s *terminal.Session) float32 { return row(t, tt, a, s).X }
	if at(m1)-at(lead) != memberIndent || at(nested) != at(m1) || at(orphan) != at(lead) {
		t.Fatalf("rows at %v, %v, %v, %v", at(lead), at(m1), at(nested), at(orphan))
	}
	if mark(lead.ID) || !mark(m1.ID) || !mark(shell.ID) || !strings.Contains(a.agentLabel(m2), fmt.Sprintf("#%d", lead.ID)) {
		t.Fatal("marks of leads: the group's lead is marked, or another lead is not")
	}
}

func TestAgentsShowTimersAndHeldTexts(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	next := time.Date(2026, 11, 20, 12, 30, 0, 0, time.Local)
	a.timers[sessions[0].ID] = agentTimer{every: 30 * time.Minute, next: next}
	a.timers[sessions[1].ID] = agentTimer{next: next.Add(150 * time.Minute)}
	a.heldSends[sessions[2].ID] = []string{"a"} // and one with a text held
	tt.Frame()
	for _, want := range []string{"⏱ 30m · 12:30", "⏱ 15:00", "✉ 1"} {
		if !tt.HasText(want) {
			t.Errorf("%q not shown", want)
		}
	}
	row(t, tt, a, sessions[0]) // its label tells the timer
	if label := a.agentLabel(sessions[0]); !strings.Contains(label, "timer 30m · 12:30") {
		t.Errorf("label %q", label)
	}
}
