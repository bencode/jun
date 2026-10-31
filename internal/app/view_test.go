package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func testNewApp(t *testing.T) *app {
	t.Helper()
	shell := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(shell, []byte("#!/bin/sh\nexec /bin/sh -i\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", shell)
	t.Setenv("ENV", "")
	a := newApp()
	t.Cleanup(func() {
		if err := a.terminals.CloseAll(); err != nil {
			t.Error(err)
		}
	})
	return a
}

func contentLabel(a *app, v *layout.View) string {
	if session := a.sessionFor(v.ID); session != nil {
		return fmt.Sprintf("%s %d", a.text("Terminal content", "终端内容"), session.ID)
	}
	return a.title(v)
}

func click(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}

func menu(t *testing.T, tt *ui.Tester, a *app, r *layout.Region, path ...string) {
	t.Helper()
	click(t, tt, fmt.Sprintf("%s %d", a.text("Region menu", "区域菜单"), r.ID))
	if err := tt.ChooseMenuItem(path...); err != nil {
		t.Fatal(err)
	}
}

func chooseTabMenu(t *testing.T, tt *ui.Tester, a *app, view *layout.View, path ...string) {
	t.Helper()
	if err := tt.RightClick(a.title(view)); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(path...); err != nil {
		t.Fatal(err)
	}
}

func perspectiveMenu(t *testing.T, tt *ui.Tester, path ...string) {
	t.Helper()
	click(t, tt, "Perspective")
	if err := tt.ChooseMenuItem(path...); err != nil {
		t.Fatal(err)
	}
}

func testApp(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	addColumn := func() {
		p := a.state.CurrentPerspective()
		if _, err := a.state.AddColumn(p.Columns[len(p.Columns)-1].ID); err != nil {
			t.Fatal(err)
		}
	}
	open := func(region *layout.Region, kind layout.ViewKind) {
		if _, err := a.state.OpenView(region.ID, kind, "Jun"); err != nil {
			t.Fatal(err)
		}
	}
	addColumn()
	open(a.state.CurrentPerspective().Columns[1], layout.ViewTerminal)
	if _, err := a.state.AddPerspective("Review"); err != nil {
		t.Fatal(err)
	}
	addColumn()
	review := a.state.CurrentPerspective()
	open(review.Columns[0], layout.ViewTree)
	open(review.Columns[1], layout.ViewFile)
	open(review.Columns[1], layout.ViewDiff)
	open(review.Bottom, layout.ViewTerminal)
	if err := a.state.SelectPerspective(a.state.CurrentProject().Perspectives[0].ID); err != nil {
		t.Fatal(err)
	}
	open(a.state.Global, layout.ViewAgents)
	a.restoreTerminals()
	return a, ui.NewTester(a.view, 1440, 900)
}

func TestRegionOpenMoveCloseAndFocus(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	left := p.Columns[0]
	right := p.Columns[1]
	menu(t, tt, a, left, "Open View", "New Terminal")
	opened := left.Views[len(left.Views)-1]
	if len(left.Views) != 2 || left.Active != opened.ID || !tt.Focused(contentLabel(a, opened)) {
		t.Fatal("New Terminal did not select/focus its content")
	}
	chooseTabMenu(t, tt, a, opened, "Move Tab To", "Jun / Work / Bottom Region")
	if !p.BottomVisible || p.Bottom.Active != opened.ID || !tt.Focused(contentLabel(a, opened)) {
		t.Fatal("move did not reveal/focus bottom")
	}
	chooseTabMenu(t, tt, a, opened, "Move Tab To", "Global")
	if a.state.Global.Active != opened.ID || !tt.Focused(contentLabel(a, opened)) {
		t.Fatal("move to Global lost focus")
	}
	chooseTabMenu(t, tt, a, opened, "Close Tab")
	if len(a.state.Global.Views) != 1 || !tt.Focused("Agents") {
		t.Fatal("close did not focus adjacent tab")
	}
	click(t, tt, "Close "+a.title(right.Views[0]))
	if len(right.Views) != 0 || right.Active != 0 || !tt.Focused(fmt.Sprintf("Region menu %d", right.ID)) {
		t.Fatal("last close did not focus empty region")
	}
}

func TestTabDragAndKeyboardSelection(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	left := p.Columns[0]
	right := p.Columns[1]
	view := left.Views[0]
	source, ok := tt.Find(a.title(view))
	if !ok {
		t.Fatal("missing source tab")
	}
	target, ok := tt.Find(a.title(right.Views[0]))
	if !ok {
		t.Fatal("missing target tab")
	}
	x, y := source.X+12, source.Y+12
	dx, dy := target.X+12, target.Y+12
	tt.Press(x, y)
	for step := 1; step <= 4; step++ {
		f := float32(step) / 4
		tt.Move(x+(dx-x)*f, y+(dy-y)*f)
	}
	tt.Release(dx, dy)
	if len(left.Views) != 0 || len(right.Views) != 2 || right.Views[0] != view || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("drag did not move/select/focus tab")
	}
	tt.Key(ui.Ctrl, ui.KeyTab)
	if right.Active != right.Views[1].ID {
		t.Fatal("arrow key did not select next tab")
	}
	source, ok = tt.Find(a.title(view))
	if !ok {
		t.Fatal("moved tab disappeared")
	}
	target, ok = tt.Find("Drop a view here")
	if !ok {
		t.Fatal("empty region disappeared")
	}
	x, y = source.X+12, source.Y+12
	dx, dy = target.X+12, target.Y+12
	tt.Press(x, y)
	tt.Move(x+20, y+20)
	tt.Move(dx, dy)
	tt.Release(dx, dy)
	if left.Active != view.ID || len(right.Views) != 1 {
		t.Fatal("drop into empty body failed")
	}
}

func TestPerspectiveGlobalAndLanguage(t *testing.T) {
	a, tt := testApp(t)
	before, _ := tt.Find("Perspective")
	if before.X >= 56 || before.Y < 800 {
		t.Fatalf("perspective should be at the rail bottom: %+v", before)
	}
	perspectiveMenu(t, tt, "Review")
	if a.state.CurrentProject().ActivePerspective != a.state.CurrentProject().Perspectives[1].ID || !tt.HasText("Diff") {
		t.Fatal("Review did not render")
	}
	global := a.state.Global
	click(t, tt, "Hide Global")
	if a.state.GlobalVisible || tt.HasText("Agents") {
		t.Fatal("Global did not disappear")
	}
	click(t, tt, "Show Global")
	if !a.state.GlobalVisible || a.state.Global != global || !tt.HasText("Agents") {
		t.Fatal("Global did not restore")
	}
	a.state.Chinese = true
	tt.Frame()
	if a.state.CurrentPerspective().Name != "Review" || !tt.HasText("布局预览") {
		t.Fatal("Chinese labels missing")
	}
	menu(t, tt, a, a.state.CurrentPerspective().Columns[0], "打开视图", "新建终端")
	if a.state.CurrentPerspective().Columns[0].Views[1].Kind != layout.ViewTerminal {
		t.Fatal("translated action failed")
	}
}

func TestEmptyStartupAndColumnControls(t *testing.T) {
	empty := testNewApp(t)
	tt := ui.NewTester(empty.view, 1000, 640)
	if !tt.HasText("Open Project…") || len(empty.state.Projects) != 0 {
		t.Fatal("startup is not an empty project picker")
	}
	click(t, tt, "Perspective")
	if tt.Menu() != nil {
		t.Fatal("Perspective should be disabled without a project")
	}
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	menu(t, tt, a, p.Columns[0], "Add Column to Right")
	if len(p.Columns) != 3 {
		t.Fatal("column not added")
	}
	// Global remains reachable even when each column has many overflowing tabs.
	for i := 0; i < 12; i++ {
		a.newTerminal(p.Columns[2].ID)
	}
	p.ColumnSplits[1] = 99
	tt.Frame()
	click(t, tt, "Hide Global")
	click(t, tt, "Show Global")
	p.ColumnSplits[1] = 50
	tt.Frame()
	menu(t, tt, a, p.Columns[2], "Remove This Column")
	if len(p.Columns) != 2 || len(p.Columns[1].Views) != 13 {
		t.Fatal("column removal lost tabs")
	}
	for _, v := range p.Columns[1].Views {
		if v.ID == p.Columns[1].Active && !tt.Focused(contentLabel(a, v)) {
			t.Fatal("removed column did not focus its destination")
		}
	}
}

func TestColumnMenusSurviveNarrowSplitsAndWindowResize(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	if _, err := a.state.AddColumn(p.Columns[len(p.Columns)-1].ID); err != nil {
		t.Fatal(err)
	}
	a.state.RailExpanded = true
	a.state.CenterPercent = 1
	p.ColumnSplits = []float32{1, 99}
	tt.Frame()
	tt.SetSize(1000, 640)
	divider, ok := tt.Find("Divider")
	if !ok {
		t.Fatal("missing column divider")
	}
	tt.Press(divider.X, 100)
	tt.Move(0, 100)
	tt.Release(0, 100)
	for _, r := range p.Columns {
		previous := len(r.Views)
		menu(t, tt, a, r, "Open View", "New Terminal")
		if len(r.Views) != previous+1 {
			t.Fatal("narrow region menu did not create a View")
		}
	}
	menu(t, tt, a, a.state.Global, "Open View", "New Terminal")
	for len(p.Columns) > 1 {
		menu(t, tt, a, p.Columns[len(p.Columns)-1], "Remove This Column")
	}
	a.state.CenterPercent = 1
	tt.Frame()
	menu(t, tt, a, p.Columns[0], "Open View", "New Terminal")
	regionMenu, ok := tt.Find(fmt.Sprintf("Region menu %d", p.Columns[0].ID))
	if !ok {
		t.Fatal("missing region menu")
	}
	globalToggle, ok := tt.Find("Hide Global")
	if !ok || regionMenu.X+regionMenu.W > globalToggle.X {
		t.Fatal("Global toggle overlaps region menu")
	}
	perspectiveMenu(t, tt, "Review")
	if a.state.CurrentProject().ActivePerspective != a.state.CurrentProject().Perspectives[1].ID {
		t.Fatal("narrow perspective selector did not work")
	}
	click(t, tt, "Hide Global")
	click(t, tt, "Show Global")
}
