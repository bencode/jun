package app

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestOpenViewEntrypoints(t *testing.T) {
	for _, position := range []string{"column", "bottom", "global"} {
		for _, entry := range []string{"empty", "region menu"} {
			t.Run(position+"/"+entry, func(t *testing.T) {
				a := testNewApp(t)
				if err := a.state.AddProject("Jun", t.TempDir()); err != nil {
					t.Fatal(err)
				}
				a.restoreTerminals()
				p := a.state.CurrentPerspective()
				r := a.state.Global
				if position == "column" {
					id, err := a.state.AddColumn(p.Columns[0].ID)
					if err != nil {
						t.Fatal(err)
					}
					r = a.state.Region(id)
				}
				if position == "bottom" {
					r = p.Bottom
					p.BottomVisible = true
				}
				if position == "global" {
					a.state.GlobalVisible = true
				}
				a.restoreConfig(filepath.Join(t.TempDir(), "config.json"))
				tt := ui.NewTester(a.view, 1440, 900)
				choose := func(label string) {
					t.Helper()
					if entry == "empty" {
						click(t, tt, "Open View")
						if err := tt.ChooseMenuItem(label); err != nil {
							t.Fatal(err)
						}
					} else {
						menu(t, tt, a, r, "Open View", label)
					}
				}
				click(t, tt, "Open View")
				tt.CloseMenu()
				tt.Advance(400 * time.Millisecond)
				if len(r.Views) != 0 || len(a.bindings) != 1 || a.persistence.dirty {
					t.Fatal("canceling menu changed layout or created a session")
				}
				choose("New Terminal")
				opened := r.Views[0]
				if len(a.bindings) != 2 || !tt.Focused(contentLabel(a, opened)) {
					t.Fatal("entry did not create and focus a real terminal")
				}
				tt.Type("echo OPEN_VIEW_INPUT")
				tt.Key(0, ui.KeyEnter)
				terminal := a.sessionFor(opened.ID).Terminal
				waitTerminal(t, tt, func() bool { return strings.Contains(terminal.Text(), "\nOPEN_VIEW_INPUT\n") })
				for _, option := range []struct {
					label  string
					kind   layout.ViewKind
					source string
				}{
					{"Agents (Preview)", layout.ViewAgents, ""}, {"Files (Preview)", layout.ViewTree, "Jun"},
				} {
					menu(t, tt, a, r, "Open View", option.label)
					view := r.Views[len(r.Views)-1]
					if view.Kind != option.kind || view.ProjectName != option.source || !tt.Focused(a.title(view)) || !tt.HasText("Layout preview") {
						t.Fatal("preview did not retain type/source or focus its tab")
					}
					menu(t, tt, a, r, "Open View", "New Terminal")
					count := len(r.Views)
					menu(t, tt, a, r, "Open View", option.label)
					if len(r.Views) != count || r.Active != view.ID {
						t.Fatal("reopening preview did not select its existing view")
					}
				}
			})
		}
	}
}

func TestPreviewSourceAndPersistence(t *testing.T) {
	a := testNewApp(t)
	path := filepath.Join(t.TempDir(), "config.json")
	a.restoreConfig(path)
	a.projectForm.name, a.projectForm.path = "A", t.TempDir()
	a.submitProject()
	a.state.GlobalVisible = true
	tt := ui.NewTester(a.view, 1440, 900)
	global := a.state.Global
	menu(t, tt, a, global, "Open View", "Files (Preview)")
	fromA := global.Views[0]
	menu(t, tt, a, global, "Open View", "Agents (Preview)")
	agents := global.Active
	a.projectForm.name, a.projectForm.path = "B", t.TempDir()
	a.submitProject()
	tt.Frame()
	menu(t, tt, a, global, "Open View", "Files (Preview)")
	fromB := global.Views[len(global.Views)-1]
	if fromA.ProjectName != "A" || fromB.ProjectName != "B" || fromA.ID == fromB.ID {
		t.Fatal("project switch rebound the Files preview")
	}
	menu(t, tt, a, global, "Open View", "Agents (Preview)")
	if len(global.Views) != 3 || global.Active != agents {
		t.Fatal("Agents preview acquired a project-specific identity")
	}
	a.selectProject("A")
	tt.Frame()
	menu(t, tt, a, global, "Open View", "Files (Preview)")
	if len(global.Views) != 3 || global.Active != fromA.ID {
		t.Fatal("returning to A did not select its Files preview")
	}
	target := a.state.CurrentPerspective().Columns[0]
	a.moveView(fromA.ID, target.ID, 0)
	tt.Frame()
	menu(t, tt, a, target, "Open View", "Files (Preview)")
	if target.Active != fromA.ID || len(target.Views) != 2 {
		t.Fatal("moving preview lost its identity")
	}
	menu(t, tt, a, global, "Open View", "Files (Preview)")
	if global.Active == fromA.ID || len(global.Views) != 3 {
		t.Fatal("different regions did not create independent views")
	}
	tt.Advance(400 * time.Millisecond)
	restored := testNewApp(t)
	restored.restoreConfig(path)
	if restored.errorOpen || restored.state.ActiveProject != "A" || restored.state.Global.Active != global.Active {
		t.Fatal("preview layout failed to restore")
	}
	restoredA := restored.state.CurrentPerspective().Columns[0]
	if restoredA.Active != fromA.ID || restoredA.Views[1].ProjectName != "A" {
		t.Fatal("moved preview source or selection did not survive restore")
	}
	if views := restored.state.Global.Views; len(views) != 3 || views[0].ID != agents || views[1].ID != fromB.ID || views[1].ProjectName != "B" || views[2].ProjectName != "A" {
		t.Fatal("mixed preview types and sources did not survive restore")
	}
}

func TestOpenViewWithoutProject(t *testing.T) {
	a := testNewApp(t)
	a.state.GlobalVisible = true
	tt := ui.NewTester(a.view, 1000, 640)
	click(t, tt, "Open View")
	for _, label := range []string{"New Terminal", "Files (Preview)"} {
		if !slices.Contains(tt.Menu(), label) {
			t.Fatalf("%s was missing without a project", label)
		}
		if err := tt.ChooseMenuItem(label); err == nil {
			t.Fatalf("%s was enabled without a project", label)
		}
	}
	if err := tt.ChooseMenuItem("Agents (Preview)"); err != nil {
		t.Fatal(err)
	}
	global := a.state.Global
	if len(global.Views) != 1 || global.Views[0].Kind != layout.ViewAgents || global.Views[0].ProjectName != "" || len(a.bindings) != 0 {
		t.Fatal("Agents preview required a project or started a shell")
	}
	a.chinese = true
	tt.Frame()
	menu(t, tt, a, global, "打开视图", "Agents（预览）")
	if len(global.Views) != 1 || !tt.HasText("布局预览") {
		t.Fatal("translated preview entry did not reuse its view")
	}
}
