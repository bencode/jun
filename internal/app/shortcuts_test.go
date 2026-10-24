package app

import (
	"fmt"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestShortcutsTargetFocusedRegionAndRespectModals(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	for _, r := range []*layout.Region{p.Columns[0], p.Bottom, a.state.Global} {
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
