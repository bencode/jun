package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestOpenViewEntrypoints(t *testing.T) {
	for _, position := range []string{"column", "bottom"} {
		for _, entry := range []string{"empty", "region menu"} {
			t.Run(position+"/"+entry, func(t *testing.T) {
				a := testNewApp(t)
				if err := a.state.AddProject("Jun", t.TempDir()); err != nil {
					t.Fatal(err)
				}
				a.restoreTerminals()
				p := a.state.CurrentPerspective()
				r := p.Bottom
				if position == "column" {
					id, err := a.state.AddColumn(p.Columns[0].ID)
					if err != nil {
						t.Fatal(err)
					}
					r = a.state.Region(id)
				}
				if position == "bottom" {
					p.BottomVisible = true
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
				// Agents and Files live in Global and the side region now.
				for _, label := range []string{"Agents", "Files", "Files (Preview)"} {
					click(t, tt, fmt.Sprintf("Region menu %d", r.ID))
					if err := tt.ChooseMenuItem("Open View", label); err == nil {
						t.Fatalf("Open View offered %s", label)
					}
					tt.CloseMenu()
				}
			})
		}
	}
}

func TestFixedRegionsKeepTheirViews(t *testing.T) {
	a, tt := testApp(t)
	p := a.state.CurrentPerspective()
	perspectiveMenu(t, tt, "Show Side Bar")
	if !p.SideVisible {
		t.Fatal("Perspective menu did not show the side region")
	}
	for _, label := range []string{"Files", "Sessions", "Skills", "Agents"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("fixed tab %s is missing", label)
		}
		if _, ok := tt.Find("Close " + label); ok {
			t.Fatalf("fixed tab %s can be closed", label)
		}
	}
	click(t, tt, "Sessions")
	if p.Side.Active != p.Side.Views[1].ID || !tt.HasText("Arrives in a later iteration.") {
		t.Fatal("clicking a side tab did not show its placeholder")
	}
	for r, item := range map[*layout.Region]string{p.Side: "Hide Side Bar", a.state.Global: "Hide Global"} {
		click(t, tt, fmt.Sprintf("Region menu %d", r.ID))
		if items := tt.Menu(); !slices.Equal(items, []string{item}) {
			t.Fatalf("region %d menu offers %v", r.ID, items)
		}
		tt.CloseMenu()
	}
	if err := tt.RightClick(a.title(p.Columns[0].Views[0])); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Move Tab To", "Global"); err == nil {
		t.Fatal("a tab moved into Global")
	}
	tt.CloseMenu()
	a.focusRegion(p.Side.ID)
	tt.Frame()
	menu(t, tt, a, p.Side, "Hide Side Bar")
	if p.SideVisible || a.targetRegion() != p.Columns[0] {
		t.Fatal("hiding the focused side region did not focus the first column")
	}
}

func TestGlobalShowsAgentsWithoutProject(t *testing.T) {
	a := testNewApp(t)
	a.state.GlobalVisible = true
	tt := ui.NewTester(a.view, 1000, 640)
	if !tt.HasText("No agents running") {
		t.Fatal("Global did not show Agents")
	}
	click(t, tt, fmt.Sprintf("Region menu %d", a.state.Global.ID))
	if items := tt.Menu(); !slices.Equal(items, []string{"Hide Global"}) {
		t.Fatalf("Global menu offers %v", items)
	}
	tt.CloseMenu()
	a.state.Chinese = true
	tt.Frame()
	if !tt.HasText("没有正在运行的 agent") || len(a.state.Global.Views) != 1 || len(a.bindings) != 0 {
		t.Fatal("Agents did not translate or started a shell")
	}
}
