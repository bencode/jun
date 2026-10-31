package app

import (
	"strings"
	"testing"

	"github.com/bencode/jun/internal/agent"
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

func TestAgentsListNarrowAndWallWide(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	idle, working, waiting := row(t, tt, a, sessions[0]), row(t, tt, a, sessions[1]), row(t, tt, a, sessions[2])
	if !(waiting.Y < working.Y && working.Y < idle.Y) || len(a.previews) != 0 {
		t.Fatal("narrow Global did not list agents waiting first")
	}
	for _, count := range []string{"All 3", "Waiting 1", "Working 1", "Idle 1"} {
		if !tt.HasText(count) {
			t.Fatalf("missing count %q", count)
		}
	}
	agents := a.state.Global.Views[0].ID
	a.moveView(agents, a.state.CurrentPerspective().Columns[0].ID, 0)
	a.state.GlobalVisible = false
	a.state.CurrentPerspective().ColumnSplits = []float32{90}
	tt.Frame()
	tt.Frame()
	if len(a.previews) != 3 {
		t.Fatalf("wide column did not show cards with previews: %d", len(a.previews))
	}
	row(t, tt, a, sessions[2])
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

func TestAgentClickReopensClosedTabInItsRegion(t *testing.T) {
	a, tt, _ := agentsApp(t)
	hidden, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.agents[hidden.ID] = agentInfo{agent.Claude, agent.Waiting}
	tt.Frame()
	global := a.state.Global
	click(t, tt, a.agentLabel(hidden))
	view := global.Views[len(global.Views)-1]
	if len(global.Views) != 2 || global.Active != view.ID || a.bindings[view.ID] != hidden.ID ||
		view.ProjectName != "Jun" || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("click did not reopen the agent's session in the Agents region")
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
