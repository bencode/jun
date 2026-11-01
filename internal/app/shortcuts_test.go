package app

import (
	"fmt"
	"slices"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestShortcutsTargetFocusedRegionAndRespectModals(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	for _, r := range []*layout.Region{p.Columns[0], p.Bottom} {
		if r == p.Bottom {
			p.BottomVisible = true
		}
		a.focusRegion(r.ID)
		tt.Frame()
		before := len(r.Views)
		tt.Key(ui.Super, ui.KeyT)
		if len(r.Views) != before+1 {
			t.Fatal("new terminal ignored active region")
		}
		opened := r.Views[len(r.Views)-1]
		session := a.sessionFor(opened.ID)
		if !tt.Focused(contentLabel(a, opened)) {
			t.Fatal("new terminal did not focus content")
		}
		tt.Key(ui.Super, ui.KeyT)
		other := r.Active
		tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
		if r.Active != opened.ID {
			t.Fatal("previous tab failed")
		}
		tt.Key(ui.Ctrl, ui.KeyTab)
		if r.Active != other {
			t.Fatal("next tab failed")
		}
		tt.Key(ui.Ctrl, ui.KeyTab)
		if r.Active != r.Views[0].ID {
			t.Fatal("next tab did not wrap")
		}
		tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
		if r.Active != other {
			t.Fatal("previous tab did not wrap")
		}
		tt.Key(ui.Super, ui.KeyW)
		tt.Key(ui.Super, ui.KeyW)
		if len(r.Views) != before || a.sessionFor(opened.ID) != nil {
			t.Fatal("close did not unbind view")
		}
		select {
		case <-session.Terminal.Done():
			t.Fatal("close shortcut terminated session")
		default:
		}
	}
	// From Global and the side region, ⌘T opens where tabs were last
	// focused and ⌘W closes nothing.
	p.SideVisible = true
	for _, r := range []*layout.Region{a.state.Global, p.Side} {
		a.focusRegion(p.Columns[1].ID)
		tt.Frame()
		a.focusRegion(r.ID)
		tt.Frame()
		views, before := slices.Clone(r.Views), len(p.Columns[1].Views)
		tt.Key(ui.Super, ui.KeyW)
		tt.Key(ui.Super, ui.KeyT)
		if !slices.Equal(r.Views, views) || len(p.Columns[1].Views) != before+1 {
			t.Fatalf("region %d: ⌘W closed a fixed view or ⌘T missed the last column", r.ID)
		}
	}
	a.errorOpen = true
	tt.Frame()
	region := a.targetRegion()
	before := len(region.Views)
	tt.Key(ui.Super, ui.KeyT)
	tt.Key(ui.Super, ui.KeyW)
	if len(region.Views) != before {
		t.Fatal("shortcut crossed modal boundary")
	}
	click(t, tt, "OK")
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	a.toggleGlobal()
	tt.Frame()
	if a.targetRegion() != p.Columns[0] {
		t.Fatal("hidden Global remained command target")
	}
}

func TestShortcutsOnEmptyRegionAndNoProject(t *testing.T) {
	a := testNewApp(t)
	tt := ui.NewTester(a.view, 1000, 640)
	tt.Key(ui.Super, ui.KeyT)
	tt.Key(ui.Super, ui.KeyW)
	tt.Key(ui.Ctrl, ui.KeyTab)
	if len(a.bindings) != 0 {
		t.Fatal("empty startup created terminal")
	}
	a.projectForm.name, a.projectForm.path = "Jun", t.TempDir()
	a.submitProject()
	tt.Frame()
	r := a.state.CurrentPerspective().Columns[0]
	tt.Key(ui.Super, ui.KeyW)
	tt.Key(ui.Super, ui.KeyW)
	tt.Key(ui.Ctrl|ui.Shift, ui.KeyTab)
	if len(r.Views) != 0 || !tt.Focused(fmt.Sprintf("Region menu %d", r.ID)) {
		t.Fatal("last close did not retain empty region")
	}
	tt.Key(ui.Super, ui.KeyT)
	if len(r.Views) != 1 {
		t.Fatal("empty region could not create terminal")
	}
}

func TestTabShortcutsSelectByPositionAndWrap(t *testing.T) {
	a, tt := testApp(t)
	r := a.state.CurrentPerspective().Columns[0]
	a.focusRegion(r.ID)
	tt.Frame()
	for len(r.Views) < 3 {
		tt.Key(ui.Super, ui.KeyT)
	}
	first, last := r.Views[0].ID, r.Views[len(r.Views)-1].ID
	steps := []struct {
		mods ui.Modifiers
		key  ui.Key
		want layout.ViewID
	}{
		{ui.Super, ui.Key1, first},
		{ui.Super | ui.Shift, ui.KeyBracketLeft, last},
		{ui.Super | ui.Shift, ui.KeyBracketRight, first},
		{ui.Super, ui.Key9, last},
		{ui.Super, ui.Key5, last},
	}
	for _, s := range steps {
		tt.Key(s.mods, s.key)
		if r.Active != s.want {
			t.Fatalf("%v+%v selected %d, want %d", s.mods, s.key, r.Active, s.want)
		}
	}
}

func TestRegionShortcutsFollowLayout(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	p.BottomVisible, p.SideVisible, a.state.GlobalVisible = true, true, true
	side, c1, c2, bottom, global := p.Side, p.Columns[0], p.Columns[1], p.Bottom, a.state.Global
	a.focusRegion(c1.ID)
	tt.Frame()
	press := func(mods ui.Modifiers, key ui.Key, want *layout.Region) {
		t.Helper()
		tt.Key(mods, key)
		if got := a.targetRegion(); got != want {
			t.Fatalf("%v+%v focused region %d, want %d", mods, key, got.ID, want.ID)
		}
	}
	arrow := ui.Super | ui.Alt
	press(arrow, ui.KeyLeft, side)
	press(arrow, ui.KeyLeft, side)
	press(arrow, ui.KeyDown, side)
	press(arrow, ui.KeyRight, c1)
	press(arrow, ui.KeyRight, c2)
	press(arrow, ui.KeyRight, global)
	press(arrow, ui.KeyRight, global)
	press(arrow, ui.KeyLeft, c2)
	press(arrow, ui.KeyDown, bottom)
	press(arrow, ui.KeyUp, c1)
	press(arrow, ui.KeyDown, bottom)
	press(arrow, ui.KeyLeft, side)
	press(arrow, ui.KeyRight, c1)
	press(arrow, ui.KeyDown, bottom)
	press(arrow, ui.KeyRight, global)
	for _, want := range []*layout.Region{side, c1, c2, bottom, global, side} {
		press(ui.Super, ui.KeyBracketRight, want)
	}
	press(ui.Super, ui.KeyBracketLeft, global)
	a.setBottomVisible(bottom.ID, false)
	press(ui.Super, ui.KeyBracketLeft, c2)
	press(ui.Super, ui.KeyBracketRight, global)
	press(arrow, ui.KeyLeft, c2)
	press(arrow, ui.KeyDown, c2)
	// ⌘B hides the focused side region, focusing the first column, and shows it again leaving focus.
	a.focusRegion(side.ID)
	tt.Frame()
	press(ui.Super, ui.KeyB, c1)
	if p.SideVisible {
		t.Fatal("⌘B did not hide the side region")
	}
	press(ui.Super, ui.KeyB, c1)
	if !p.SideVisible {
		t.Fatal("⌘B did not show the side region")
	}
}

func TestPerspectiveAndProjectShortcuts(t *testing.T) {
	empty := testNewApp(t)
	et := ui.NewTester(empty.view, 1000, 640)
	for _, k := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{ui.Super | ui.Alt, ui.KeyBracketRight}, {ui.Super | ui.Alt, ui.Key1}, {ui.Ctrl | ui.Super, ui.KeyBracketRight},
		{ui.Ctrl | ui.Super, ui.Key1}, {ui.Super | ui.Shift, ui.KeyN}, {ui.Super | ui.Alt, ui.KeyRight}, {ui.Super, ui.KeyBracketRight}} {
		et.Key(k.mods, k.key)
	}
	if empty.state.CurrentProject() != nil || empty.perspectiveForm.open || empty.errorOpen {
		t.Fatal("shortcuts acted without a project")
	}

	a, tt := testApp(t)
	project := a.state.CurrentProject()
	work, review := project.Perspectives[0].ID, project.Perspectives[1].ID
	tt.Key(ui.Super|ui.Alt, ui.KeyBracketRight)
	if project.ActivePerspective != review || a.targetRegion() != a.state.CurrentPerspective().Columns[0] {
		t.Fatal("next perspective did not select and focus Review")
	}
	tt.Key(ui.Super|ui.Alt, ui.KeyBracketRight)
	tt.Key(ui.Super|ui.Alt, ui.Key9)
	if project.ActivePerspective != work {
		t.Fatal("perspective did not wrap or out-of-range position changed it")
	}
	tt.Key(ui.Super|ui.Alt, ui.Key2)
	if project.ActivePerspective != review {
		t.Fatal("perspective position did not select Review")
	}

	if err := a.state.AddProject("Other", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := a.state.SelectProject("Jun"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl|ui.Super, ui.KeyBracketRight)
	if a.state.ActiveProject != "Other" {
		t.Fatal("next project did not select Other")
	}
	tt.Key(ui.Ctrl|ui.Super, ui.KeyBracketRight)
	tt.Key(ui.Ctrl|ui.Super, ui.Key9)
	if a.state.ActiveProject != "Jun" || project.ActivePerspective != review {
		t.Fatal("project did not wrap, ignore out-of-range position or keep its perspective")
	}
	tt.Key(ui.Ctrl|ui.Super, ui.Key2)
	tt.Key(ui.Ctrl|ui.Super, ui.KeyBracketLeft)
	if a.state.ActiveProject != "Jun" {
		t.Fatal("project position or previous failed")
	}

	tt.Key(ui.Super|ui.Shift, ui.KeyN)
	if !a.perspectiveForm.open || a.perspectiveForm.id != 0 {
		t.Fatal("new perspective shortcut did not open the form")
	}
}
