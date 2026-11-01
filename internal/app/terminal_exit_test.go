package app

import (
	"fmt"
	"testing"

	"github.com/bencode/jun/internal/layout"
)

func TestExitSelectsNeighborAndEmptyRegion(t *testing.T) {
	a, tt := testApp(t)
	r := a.state.CurrentPerspective().Columns[0]
	left := r.Views[0]
	a.newTerminal(r.ID)
	middle := r.Views[1]
	a.newTerminal(r.ID)
	right := r.Views[2]
	if err := a.state.SelectView(r.ID, middle.ID); err != nil {
		t.Fatal(err)
	}
	a.focusRegion(r.ID)
	tt.Frame()
	for _, pair := range [][2]*layout.View{{middle, right}, {right, left}, {left, nil}} {
		session := a.sessionFor(pair[0].ID)
		session.Terminal.Send([]byte("false; exit\n"))
		waitTerminal(t, tt, func() bool { return a.terminals.Get(session.ID) == nil })
		tt.Frame()
		if pair[1] != nil {
			if r.Active != pair[1].ID || !tt.Focused(contentLabel(a, pair[1])) {
				t.Fatal("exit did not focus the adjacent surviving tab")
			}
		} else if len(r.Views) != 0 || !tt.Focused(fmt.Sprintf("Region menu %d", r.ID)) {
			t.Fatal("last exit did not preserve and focus the empty region")
		}
	}
	if len(a.hiddenTerminals("Jun")) != 0 {
		t.Fatal("exited sessions can be reopened")
	}
}

func TestBackgroundExitDoesNotChangeCurrentWork(t *testing.T) {
	for _, location := range []string{"column", "inactive tab", "bottom", "perspective", "project"} {
		t.Run(location, func(t *testing.T) {
			a, tt := testApp(t)
			p := a.state.CurrentPerspective()
			r := p.Columns[0]
			background := p.Columns[1].Views[0]
			session := a.sessionFor(background.ID)
			switch location {
			case "inactive tab":
				a.moveView(background.ID, r.ID, 0)
				if err := a.state.SelectView(r.ID, r.Views[0].ID); err != nil {
					t.Fatal(err)
				}
			case "bottom":
				a.moveView(background.ID, p.Bottom.ID, 0)
				p.BottomVisible = false
			case "perspective":
				a.selectPerspective(a.state.CurrentProject().Perspectives[1].ID)
				r = a.state.CurrentPerspective().Columns[0]
			case "project":
				if err := a.state.AddProject("Other", t.TempDir()); err != nil {
					t.Fatal(err)
				}
				a.restoreTerminals()
				r = a.state.CurrentPerspective().Columns[0]
			}
			a.focusRegion(r.ID)
			tt.Frame()
			active := r.Active
			project := a.state.ActiveProject
			perspective := a.state.CurrentPerspective()
			global, bottom := a.state.GlobalVisible, p.BottomVisible
			session.Terminal.Send([]byte("exit 0\n"))
			waitTerminal(t, tt, func() bool { return a.terminals.Get(session.ID) == nil })
			tt.Frame()
			if r.Active != active || a.state.ActiveProject != project || a.state.CurrentPerspective() != perspective ||
				a.state.GlobalVisible != global || p.BottomVisible != bottom || !tt.Focused(contentLabel(a, r.Views[0])) {
				t.Fatal("background exit disturbed the current work")
			}
		})
	}
}

func TestExitedHiddenSessionCannotBeReopened(t *testing.T) {
	a, tt := testApp(t)
	r := a.state.CurrentPerspective().Columns[0]
	view := r.Views[0]
	session := a.sessionFor(view.ID)
	a.closeView(r.ID, view.ID)
	if len(a.hiddenTerminals("Jun")) != 1 {
		t.Fatal("manual close did not preserve the running session")
	}
	session.Terminal.Send([]byte("false; exit\n"))
	waitTerminal(t, nil, func() bool {
		select {
		case <-session.Terminal.Done():
			return true
		default:
			return false
		}
	})
	if len(a.hiddenTerminals("Jun")) != 0 {
		t.Fatal("exit offered for reopening before the next frame")
	}
	tt.Frame()
	if a.terminals.Get(session.ID) != nil {
		t.Fatal("hidden session was not released")
	}
}
