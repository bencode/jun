package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestPerspectiveFormCreatesEmptyAndRenamesLiveTerminal(t *testing.T) {
	a, tt := testApp(t)
	work := a.state.CurrentPerspective()
	count := len(a.bindings)
	perspectiveMenu(t, tt, "New Perspective…")
	if !tt.Focused("Perspective name") {
		t.Fatal("new perspective did not focus its name")
	}
	tt.Key(ui.Super, ui.KeyT)
	tt.Key(ui.Super, ui.KeyW)
	if len(a.bindings) != count {
		t.Fatal("terminal shortcuts crossed the perspective modal")
	}
	// The suggestion is selected, so one Backspace clears it.
	if a.perspectiveForm.name != "Monitor" {
		t.Fatalf("suggested %q beside Work and Review", a.perspectiveForm.name)
	}
	tt.Key(0, ui.KeyBackspace)
	click(t, tt, "Create")
	if !a.perspectiveForm.open || a.perspectiveForm.error == "" {
		t.Fatal("blank name was accepted")
	}
	click(t, tt, "Perspective name")
	tt.Type(" work ")
	tt.Key(0, ui.KeyEnter)
	if !a.perspectiveForm.open || a.perspectiveForm.error == "" || a.state.CurrentPerspective() != work {
		t.Fatal("duplicate name changed perspective")
	}
	tt.Key(ui.Super, ui.KeyA)
	tt.Type("  Investigation  ")
	tt.Key(0, ui.KeyEnter)
	p := a.state.CurrentPerspective()
	if a.perspectiveForm.open || p.Name != "Investigation" || len(p.Columns) != 1 || p.BottomVisible ||
		len(p.Columns[0].Views) != 0 || len(a.bindings) != count || len(a.state.CurrentProject().Perspectives) != 3 {
		t.Fatal("creation was not an empty, selected perspective")
	}
	if !tt.Focused(fmt.Sprintf("Region menu %d", p.Columns[0].ID)) {
		t.Fatal("empty perspective did not focus its region menu")
	}
	tt.Key(ui.Super, ui.KeyT)
	view := p.Columns[0].Views[0]
	session := a.sessionFor(view.ID)
	perspectiveMenu(t, tt, "Rename Perspective…")
	if a.perspectiveForm.name != "Investigation" || !tt.Focused("Perspective name") {
		t.Fatal("rename did not prefill the current name")
	}
	tt.Key(ui.Super, ui.KeyA)
	tt.Type("设计审阅")
	tt.Key(0, ui.KeyEnter)
	if a.state.CurrentPerspective() != p || p.Name != "设计审阅" || a.sessionFor(view.ID) != session ||
		!tt.Focused(contentLabel(a, view)) || len(a.bindings) != count+1 {
		t.Fatal("rename replaced content, session or focus")
	}
	perspectiveMenu(t, tt, "Work")
	if a.state.CurrentPerspective() != work {
		t.Fatal("could not switch back to the original perspective")
	}
	perspectiveMenu(t, tt, "设计审阅")
	if a.sessionFor(view.ID) != session || !tt.Focused(contentLabel(a, view)) {
		t.Fatal("switching recreated the terminal or lost focus")
	}
}

func TestPerspectiveCancelReturnsToCommandTarget(t *testing.T) {
	a, tt := testApp(t)
	right := a.state.CurrentPerspective().Columns[1]
	a.focusRegion(right.ID)
	tt.Frame()
	count := len(a.state.CurrentProject().Perspectives)
	perspectiveMenu(t, tt, "New Perspective…")
	tt.Type("Discard me")
	click(t, tt, "Cancel")
	if a.perspectiveForm.open || len(a.state.CurrentProject().Perspectives) != count ||
		a.targetRegion() != right || !tt.Focused(contentLabel(a, right.Views[0])) {
		t.Fatal("cancel changed layout or failed to restore focus")
	}
	perspectiveMenu(t, tt, "Rename Perspective…")
	tt.Key(ui.Super, ui.KeyA)
	tt.Type("Discard rename")
	tt.Key(0, ui.KeyEscape)
	if a.perspectiveForm.open || a.state.CurrentPerspective().Name != "Work" || !tt.Focused(contentLabel(a, right.Views[0])) {
		t.Fatal("Escape applied a rename or lost focus")
	}
}

func TestRailSettingsAndPerspectiveContext(t *testing.T) {
	a, tt := testApp(t)
	originalMenu := mygo.App.Menu()
	t.Cleanup(func() { mygo.App.SetMenu(originalMenu) })
	path := filepath.Join(t.TempDir(), "config.json")
	a.restoreConfig(path)
	perspective, ok := tt.Find("Perspective")
	settings, found := tt.Find("Settings")
	if !ok || !found || perspective.X >= 56 || settings.X >= 56 || perspective.Y >= settings.Y || settings.Y < 850 {
		t.Fatal("rail controls are not fixed at its bottom")
	}
	click(t, tt, "Settings")
	if err := tt.ChooseMenuItem("Language", "简体中文"); err != nil {
		t.Fatal(err)
	}
	if !a.state.Chinese || a.state.CurrentPerspective().Name != "Work" || !tt.HasText("设置") {
		t.Fatal("language switch failed or translated user data")
	}
	if err := a.saveLayout(); err != nil {
		t.Fatal(err)
	}
	if saved, err := config.Load(path); err != nil || !saved.Chinese {
		t.Fatalf("language was not saved: %v", err)
	}
	if !slices.ContainsFunc(mygo.App.Menu().Items(), func(item *mygo.MenuItem) bool { return item.Label == "语言" }) {
		t.Fatal("settings did not refresh the native application menu")
	}
	click(t, tt, "展开或收起项目名称")
	tt.Frame()
	if !tt.HasText("Work") || a.projectRailWidth() != 192 {
		t.Fatal("expanded rail did not show the current perspective")
	}
	longName := "设计文档与工作流程的长名称 Perspective"
	if err := a.state.RenamePerspective(a.state.CurrentPerspective().ID, longName); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	button, _ := tt.Find("视角")
	if button.X+button.W > 192 {
		t.Fatal("long name escaped the rail")
	}
	click(t, tt, "设置")
	if err := tt.ChooseMenuItem("语言", "English"); err != nil {
		t.Fatal(err)
	}
	if a.state.Chinese || a.state.CurrentPerspective().Name != longName {
		t.Fatal("language switch changed a perspective name")
	}
	if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	click(t, tt, "Perspective")
	if err := tt.ChooseMenuItem(longName); err == nil {
		t.Fatal("another project's perspective appeared in the menu")
	}
	if err := tt.ChooseMenuItem("Brain · Perspectives"); err == nil {
		t.Fatal("project context heading should be disabled")
	}
	tt.CloseMenu()
}

func TestDeletePerspectiveClosesItsTerminals(t *testing.T) {
	a, tt := testApp(t)
	project := a.state.CurrentProject()
	work, review := project.Perspectives[0], project.Perspectives[1]
	if _, err := a.state.AddPerspective("Notes"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	perspectiveMenu(t, tt, "Delete Perspective…")
	if a.confirmation.open || len(project.Perspectives) != 2 || len(a.terminals.All()) != 3 {
		t.Fatal("deleting a perspective without terminals asked or closed terminals")
	}
	a.selectPerspective(review.ID)
	tt.Frame()
	perspectiveMenu(t, tt, "Delete Perspective…")
	tt.Frame()
	if !a.confirmation.open || len(project.Perspectives) != 2 {
		t.Fatal("deleting a perspective with a terminal did not ask first")
	}
	click(t, tt, "Delete")
	if len(project.Perspectives) != 1 || project.ActivePerspective != work.ID || len(a.terminals.All()) != 2 || len(a.bindings) != 2 {
		t.Fatal("deleted perspective kept its terminal")
	}
	click(t, tt, "Perspective")
	if err := tt.ChooseMenuItem("Delete Perspective…"); err == nil {
		t.Fatal("the last perspective could be deleted")
	}
}

func TestSuggestedPerspectiveName(t *testing.T) {
	project := func(names ...string) *layout.Project {
		p := &layout.Project{}
		for _, name := range names {
			p.Perspectives = append(p.Perspectives, &layout.Perspective{Name: name})
		}
		return p
	}
	for _, c := range []struct {
		names []string
		want  string
	}{
		{[]string{"Work"}, "Review"},
		{[]string{"work", "REVIEW", "Debug"}, "Monitor"},
		{[]string{"Work", "Review", "Monitor", "Debug", "Notes"}, "Perspective 6"},
		{[]string{"Work", "Review", "Monitor", "Debug", "Notes", "Perspective 7"}, "Perspective 8"},
	} {
		if got := suggestedPerspectiveName(project(c.names...)); got != c.want {
			t.Errorf("beside %v: %q, want %q", c.names, got, c.want)
		}
	}
}
