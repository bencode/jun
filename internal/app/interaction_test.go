package app

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestRegionMenusRespectScope(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	p.BottomVisible = true
	tt.Frame()
	for _, r := range []*layout.Region{p.Columns[0], p.Bottom} {
		click(t, tt, fmt.Sprintf("Region menu %d", r.ID))
		items := tt.Menu()
		for _, name := range []string{"Open View", "Reopen Terminal"} {
			if !slices.Contains(items, name) {
				t.Fatalf("region %d is missing %s", r.ID, name)
			}
		}
		for _, name := range []string{"Close View", "Close Tab", "Move View To", "Move Tab To", "Show Bottom Region"} {
			if slices.Contains(items, name) {
				t.Fatalf("region %d contains unrelated action %s", r.ID, name)
			}
		}
		column := r == p.Columns[0]
		for _, name := range []string{"Add Column to Right", "Remove This Column"} {
			if slices.Contains(items, name) != column {
				t.Fatalf("region %d has incorrect column capability", r.ID)
			}
		}
		if slices.Contains(items, "Hide Bottom Region") != (r == p.Bottom) || slices.Contains(items, "Hide Global") || slices.Contains(items, "Hide Side Bar") {
			t.Fatal("hide action controls another region")
		}
		tt.CloseMenu()
	}
	if len(p.Columns) != 2 {
		t.Fatal("inspecting menus modified columns")
	}
}

func TestTabContextTargetsInactiveTab(t *testing.T) {
	a, tt := testApp(t)
	r := a.state.CurrentPerspective().Columns[0]
	original := r.Views[0]
	session := a.sessionFor(original.ID)
	a.newTerminal(r.ID)
	tt.Frame()
	active := r.Active
	if err := tt.RightClick(a.title(original)); err != nil {
		t.Fatal(err)
	}
	if r.Active != active {
		t.Fatal("right-click selected an inactive tab")
	}
	if err := tt.ChooseMenuItem("Close Tab"); err != nil {
		t.Fatal(err)
	}
	if r.Active != active || len(r.Views) != 1 || a.sessionFor(original.ID) != nil {
		t.Fatal("context close targeted active tab")
	}
	session.Terminal.Send([]byte("echo CLOSED_TAB_ALIVE\n"))
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nCLOSED_TAB_ALIVE\n") })
	moving := r.Views[0]
	movingSession := a.sessionFor(moving.ID)
	a.newTerminal(r.ID)
	tt.Frame()
	active = r.Active
	chooseTabMenu(t, tt, a, moving, "Move Tab To", "Jun / Work / Bottom Region")
	bottom := a.state.CurrentPerspective().Bottom
	if r.Active != active || bottom.Active != moving.ID || !a.state.CurrentPerspective().BottomVisible || a.sessionFor(moving.ID) != movingSession {
		t.Fatal("context move lost target, selection or session")
	}
	if !tt.Focused(contentLabel(a, moving)) {
		t.Fatal("moved tab did not focus its content")
	}
	a.closeView(r.ID, moving.ID)
	if !a.errorOpen || a.sessionFor(moving.ID) != movingSession || bottom.Active != moving.ID {
		t.Fatal("close accepted the wrong source region")
	}
}

func TestColumnRemovalKeepsActiveSession(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	left, right := p.Columns[0], p.Columns[1]
	view := right.Views[0]
	session := a.sessionFor(view.ID)
	menu(t, tt, a, left, "Add Column to Right")
	if len(p.Columns) != 3 || p.Columns[0] != left || p.Columns[2] != right {
		t.Fatal("column was not inserted next to anchor")
	}
	middle := p.Columns[1]
	menu(t, tt, a, middle, "Open View", "New Terminal")
	click(t, tt, fmt.Sprintf("Region menu %d", middle.ID))
	if err := tt.ChooseMenuItem("Add Column to Right"); err == nil {
		t.Fatal("fourth column was enabled")
	}
	tt.CloseMenu()
	menu(t, tt, a, right, "Remove This Column")
	if len(p.Columns) != 2 || p.Columns[1] != middle || middle.Active != view.ID || a.sessionFor(view.ID) != session || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("removal changed active content, session or focus")
	}
	tt.Type("echo REMOVED_COLUMN_ALIVE")
	tt.Key(0, ui.KeyEnter)
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nREMOVED_COLUMN_ALIVE\n") })
}

func TestPerspectiveMenuPreservesCommandTarget(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	right := p.Columns[1]
	a.focusRegion(right.ID)
	tt.Frame()
	click(t, tt, "Perspective")
	if a.targetRegion() != right {
		t.Fatal("Perspective control activated first column")
	}
	tt.CloseMenu()
	before := len(right.Views)
	tt.Key(ui.Super, ui.KeyT)
	if len(right.Views) != before+1 {
		t.Fatal("shortcut targeted the first column after menu dismissal")
	}
	perspectiveMenu(t, tt, "Work")
	if a.targetRegion() != right {
		t.Fatal("choosing current perspective reset active region")
	}
	click(t, tt, "Show Bottom Region")
	if !p.BottomVisible || a.targetRegion() != right || !tt.Focused(contentLabel(a, right.Views[len(right.Views)-1])) {
		t.Fatal("showing bottom stole focus")
	}
	a.focusRegion(p.Bottom.ID)
	tt.Frame()
	menu(t, tt, a, p.Bottom, "Hide Bottom Region")
	if p.BottomVisible || a.targetRegion() != p.Columns[0] {
		t.Fatal("hiding active bottom did not return to first column")
	}
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	click(t, tt, "Show Bottom Region")
	if a.targetRegion() != a.state.Global {
		t.Fatal("Perspective layout operation stole Global focus")
	}
	perspectiveMenu(t, tt, "Review")
	if a.targetRegion() != a.state.CurrentPerspective().Columns[0] || !p.BottomVisible {
		t.Fatal("perspective switching reset layout or kept old command target")
	}
}

func TestRemoveFirstColumnFocusesMovedTab(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	source, target := p.Columns[0], p.Columns[1]
	view := source.Views[0]
	session := a.sessionFor(view.ID)
	menu(t, tt, a, source, "Remove This Column")
	if len(p.Columns) != 1 || p.Columns[0] != target || target.Active != view.ID || !tt.Focused(contentLabel(a, view)) || a.sessionFor(view.ID) != session {
		t.Fatal("removing first column lost moved content or focus")
	}
	tt.Type("echo FIRST_COLUMN_ALIVE")
	tt.Key(0, ui.KeyEnter)
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nFIRST_COLUMN_ALIVE\n") })
}
