package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func globalApp(t *testing.T) (*app, *ui.Tester) {
	t.Helper()
	a, tt, _ := agentsApp(t)
	a.state.GlobalVisible = true
	tt.Frame()
	tt.Frame()
	return a, tt
}

func titles(t *testing.T, tt *ui.Tester) (ui.Rect, ui.Rect) {
	t.Helper()
	agents, ok := tt.Find("Agents")
	messages, shown := tt.Find("Messages")
	if !ok || !shown {
		t.Fatalf("Global shows %v", tt.Texts())
	}
	return agents, messages
}

func savedAndLoaded(t *testing.T, a *app) *layout.State {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, a.state); err != nil {
		t.Fatal(err)
	}
	s, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestGlobalArrangesItsViews(t *testing.T) {
	a, tt := globalApp(t)
	agents, messages := titles(t, tt)
	if agents.Y != messages.Y || messages.X <= agents.X {
		t.Fatalf("side by side: agents %+v, messages %+v", agents, messages)
	}
	menu(t, tt, a, a.state.Global, "Stacked")
	tt.Frame()
	agents, messages = titles(t, tt)
	if agents.X != messages.X || messages.Y <= agents.Y {
		t.Fatalf("stacked: agents %+v, messages %+v", agents, messages)
	}
	if savedAndLoaded(t, a).GlobalArrange != layout.GlobalStacked {
		t.Fatal("the layout was not saved")
	}
}

func TestGlobalKeepsTheFocusAsItsViewsChange(t *testing.T) {
	a, tt := globalApp(t)
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	tt.Frame()
	menu(t, tt, a, a.state.Global, "Stacked")
	tt.Frame()
	tt.Frame()
	if !tt.Focused("Agents") {
		t.Fatal("changing the layout lost the focus")
	}

	// Hiding the view in focus gives it to the next, which has the menu now.
	menu(t, tt, a, a.state.Global, "Agents")
	tt.Frame()
	tt.Frame()
	if !tt.Focused("Messages") || tt.HasText("No agents running") {
		t.Fatalf("Global shows %v", tt.Texts())
	}

	// The last view hides Global, and the focus leaves it.
	menu(t, tt, a, a.state.Global, "Messages")
	tt.Frame()
	tt.Frame()
	if a.state.GlobalVisible || a.activeRegion == a.state.Global.ID {
		t.Fatalf("Global visible %v, active region %d", a.state.GlobalVisible, a.activeRegion)
	}
}

func TestCycleTabSkipsHiddenViews(t *testing.T) {
	a, tt := globalApp(t)
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	tt.Frame()
	a.cycleTab(1)
	tt.Frame()
	tt.Frame()
	if !tt.Focused("Messages") || a.state.Global.Active != a.state.Global.Views[1].ID {
		t.Fatal("the shortcut did not move to Messages")
	}
	a.toggleGlobalView(a.state.Global.Views[0].ID)
	tt.Frame()
	a.cycleTab(1)
	if a.state.Global.Active != a.state.Global.Views[1].ID {
		t.Fatal("the shortcut went to a hidden view")
	}
}

func TestNumberedTabsCountTheViewsShown(t *testing.T) {
	a, tt := globalApp(t)
	messages := a.state.Global.Views[1]
	a.focusRegion(a.state.Global.ID)
	tt.Frame()
	a.toggleGlobalView(a.state.Global.Views[0].ID)
	tt.Frame()
	tt.Frame()
	a.selectTabAt(0)
	tt.Frame()
	tt.Frame()
	if a.state.Global.Active != messages.ID || a.pendingFocus != 0 {
		t.Fatalf("active %d, pending focus %d", a.state.Global.Active, a.pendingFocus)
	}
	a.selectTabAt(lastTab)
	if a.state.Global.Active != messages.ID {
		t.Fatal("the last tab was a hidden view")
	}
}

func TestDraggingBetweenGlobalsViews(t *testing.T) {
	a, tt := globalApp(t)
	_, messages := titles(t, tt)
	before := a.state.Global.Views[0].Share
	tt.Press(messages.X-1, 300)
	tt.Move(messages.X-101, 300)
	tt.Release(messages.X-101, 300)
	tt.Frame()
	share := a.state.Global.Views[0].Share
	if share >= before {
		t.Fatalf("Agents' share went from %v to %v", before, share)
	}
	if got := savedAndLoaded(t, a).Global.Views[0].Share; got != share {
		t.Fatalf("saved %v, dragged to %v", got, share)
	}
	a.toggleGlobalArrange()
	tt.Frame()
	tt.Frame()
	if a.state.Global.Views[0].Share != share {
		t.Fatal("the other layout has another share")
	}
}

func TestGlobalKeepsItsColumnsUsableAndTheirShares(t *testing.T) {
	a, tt := globalApp(t)
	resize(tt, 1000, 640)
	agents, messages := titles(t, tt)
	if messages.X-agents.X < minimumRegionWidth || messages.W < minimumRegionWidth {
		t.Fatalf("columns at %+v and %+v", agents, messages)
	}
	if a.state.Global.Views[0].Share != 60 {
		t.Fatalf("a small room changed Agents' share to %v", a.state.Global.Views[0].Share)
	}
}

func TestOpeningGlobalsMenuKeepsTheActiveView(t *testing.T) {
	a, tt := globalApp(t)
	a.selectTab(a.state.Global, a.state.Global.Views[1].ID)
	tt.Frame()
	tt.Frame()
	click(t, tt, fmt.Sprintf("Region menu %d", a.state.Global.ID))
	tt.Frame()
	tt.CloseMenu()
	tt.Frame()
	if a.state.Global.Active != a.state.Global.Views[1].ID {
		t.Fatal("the menu, on the first view, made it the active one")
	}
}

func TestADividerThatCannotMoveLeavesTheShares(t *testing.T) {
	a, tt := globalApp(t)
	resize(tt, 800, 640)
	_, messages := titles(t, tt)
	if messages.W >= minimumRegionWidth {
		t.Fatalf("Global has room: %+v", messages)
	}
	for range 2 {
		tt.Press(messages.X-1, 300)
		tt.Move(messages.X-41, 300)
		tt.Release(messages.X-41, 300)
		tt.Frame()
	}
	if a.state.Global.Views[0].Share != 60 {
		t.Fatalf("Agents' share is %v", a.state.Global.Views[0].Share)
	}
}

func TestPeekNeedsNoAgentsView(t *testing.T) {
	a, tt := globalApp(t)
	a.toggleGlobalView(a.state.Global.Views[0].ID)
	tt.Frame()
	tt.Frame()
	tt.Key(ui.Super, ui.KeyJ)
	tt.Frame()
	if a.peek.id == 0 {
		t.Fatal("⌘J peeked nothing with Agents hidden")
	}
}

func TestEveryViewOfGlobalHasACommand(t *testing.T) {
	a, tt := globalApp(t)
	commands := a.commands()
	for _, title := range []string{"Toggle Agents", "Toggle Messages", "Toggle Global Layout"} {
		if !slices.ContainsFunc(commands, func(p paletteItem) bool { return p.title == title }) {
			t.Fatalf("no command %q", title)
		}
	}
	i := slices.IndexFunc(commands, func(p paletteItem) bool { return p.title == "Toggle Messages" })
	commands[i].run()
	tt.Frame()
	if !a.state.Global.Views[1].Hidden {
		t.Fatal("the command did not hide Messages")
	}
}
