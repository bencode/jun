package app

import (
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestRouteGoesBackAndForwardAcrossTabsPerspectivesAndProjects(t *testing.T) {
	a, tt := testApp(t)
	if err := a.state.AddProject("Other", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := a.state.SelectProject("Jun"); err != nil {
		t.Fatal(err)
	}
	jun := a.state.CurrentProject()
	work, review := jun.Perspectives[0], jun.Perspectives[1]
	first, second, diff := work.Columns[0].Active, work.Columns[1].Active, review.Columns[0].Active
	a.focusRegion(work.Columns[0].ID)
	tt.Frame()
	// The terminal holds the keyboard focus, as in use: a build after a
	// jump finds it in the region left until the target takes it.
	if !tt.Focused(contentLabel(a, work.Columns[0].Views[0])) {
		t.Fatal("the first column's terminal should have the focus")
	}
	at := func(want layout.ViewID, step string) {
		t.Helper()
		if got := a.location().View; got != want {
			t.Fatalf("%s: at view %d, want %d", step, got, want)
		}
	}
	tt.Key(ui.Super|ui.Alt, ui.KeyRight)        // the second column
	tt.Key(ui.Super|ui.Alt, ui.KeyBracketRight) // Review
	tt.Key(ui.Ctrl|ui.Super, ui.KeyBracketRight)
	if a.state.ActiveProject != "Other" {
		t.Fatal("the project shortcut did not select Other")
	}
	other := a.location()

	tt.Key(ui.Super, ui.KeyBracketLeft)
	at(diff, "back from Other")
	tt.Key(ui.Super, ui.KeyBracketLeft)
	at(second, "back from Review")
	tt.Key(ui.Super, ui.KeyBracketLeft)
	at(first, "back from the second column")
	tt.Key(ui.Super, ui.KeyBracketLeft)
	at(first, "back from the first location")
	tt.Key(ui.Super, ui.KeyBracketRight)
	tt.Key(ui.Super, ui.KeyBracketRight)
	tt.Key(ui.Super, ui.KeyBracketRight)
	if a.state.ActiveProject != "Other" || !layout.SameLocation(a.location(), other) {
		t.Fatal("forward did not come back to Other")
	}
}

func TestRouteSkipsClosedTabsAndCountsAJumpOnce(t *testing.T) {
	a, tt := testApp(t)
	work, review := a.state.CurrentProject().Perspectives[0], a.state.CurrentProject().Perspectives[1]
	first, second, diff := work.Columns[0].Active, work.Columns[1].Active, review.Columns[0].Active
	a.focusRegion(work.Columns[0].ID)
	tt.Frame()
	// The side region and Global take the focus but are no place.
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	if len(a.route.history.Before) != 0 {
		t.Fatalf("focusing Global recorded %v", a.route.history.Before)
	}
	a.focusRegion(work.Columns[0].ID)
	tt.Key(ui.Super|ui.Alt, ui.KeyRight)
	tt.Key(ui.Super|ui.Alt, ui.KeyBracketRight)
	tt.Key(ui.Super, ui.KeyBracketLeft)
	tt.Key(ui.Super, ui.KeyBracketLeft) // at first; ahead: second, then diff

	// Closing a place ahead skips it going forward.
	a.closeView(work.Columns[1].ID, second)
	tt.Frame()
	tt.Key(ui.Super, ui.KeyBracketRight)
	if got := a.location().View; got != diff {
		t.Fatalf("forward went to view %d, want the diff %d past the closed tab", got, diff)
	}
	// Closing where the user is moves them, which is no move: what lay ahead stays.
	tt.Key(ui.Super, ui.KeyBracketLeft)
	a.closeView(work.Columns[0].ID, first)
	tt.Frame()
	if len(a.route.history.After) != 1 {
		t.Fatalf("closing the current tab changed what lies ahead: %v", a.route.history.After)
	}
	tt.Key(ui.Super, ui.KeyBracketRight)
	if got := a.location().View; got != diff {
		t.Fatalf("forward after closing went to view %d", got)
	}

	// Showing a terminal of another perspective switches perspective,
	// region and tab in one action: one step back undoes it.
	a.selectPerspective(work.ID)
	tt.Frame()
	before := a.location()
	bottom := review.Bottom.Active
	a.revealSession(a.bindings[bottom])
	tt.Frame()
	if a.location().View != bottom || a.state.CurrentPerspective() != review {
		t.Fatal("the terminal's tab did not show")
	}
	tt.Key(ui.Super, ui.KeyBracketLeft)
	if !layout.SameLocation(a.location(), before) || a.state.CurrentPerspective() != work {
		t.Fatalf("one step back went to %+v, want %+v", a.location(), before)
	}
}
