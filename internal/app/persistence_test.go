package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/config"
	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func persistedMatches(t *testing.T, a *app, path string) {
	t.Helper()
	saved, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	current := a.state
	if !reflect.DeepEqual(saved.Projects, current.Projects) || !reflect.DeepEqual(saved.Global, current.Global) || saved.ActiveProject != current.ActiveProject || saved.GlobalVisible != current.GlobalVisible || saved.RailExpanded != current.RailExpanded || saved.CenterPercent != current.CenterPercent {
		t.Fatal("saved layout does not match the visible state")
	}
}

func TestSaveRemovesSuccessfulExitBeforeNextFrame(t *testing.T) {
	a := testNewApp(t)
	path := filepath.Join(t.TempDir(), "config.json")
	a.restoreConfig(path)
	if err := a.state.AddProject("Jun", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	a.restoreTerminals()
	a.layoutChanged()
	if err := a.saveLayout(); err != nil {
		t.Fatal(err)
	}
	r := a.state.CurrentPerspective().Columns[0]
	session := a.sessionFor(r.Active)
	session.Terminal.Send([]byte("exit 0\n"))
	waitTerminal(t, nil, func() bool {
		select {
		case <-session.Terminal.Done():
			return true
		default:
			return false
		}
	})
	event := &mygo.CloseEvent{}
	a.beforeClose(event)
	restored := testNewApp(t)
	restored.restoreConfig(path)
	restored.restoreTerminals()
	if event.DefaultPrevented() || len(restored.state.CurrentPerspective().Columns[0].Views) != 0 || len(restored.bindings) != 0 {
		t.Fatal("saving restored a successfully exited terminal")
	}
}

func TestAutosaveWaitsForIdle(t *testing.T) {
	a, tt := testApp(t)
	path := filepath.Join(t.TempDir(), "config.json")
	a.restoreConfig(path)
	menu(t, tt, a, a.state.CurrentPerspective().Columns[0], "Open View", "New Terminal")
	tt.Advance(200 * time.Millisecond)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("saved before idle: %v", err)
	}
	click(t, tt, "Toggle project names")
	tt.Advance(200 * time.Millisecond)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("did not merge continuous changes: %v", err)
	}
	tt.Advance(100 * time.Millisecond)
	persistedMatches(t, a, path)
}

func TestLayoutUIChangesAreSaved(t *testing.T) {
	cases := map[string]func(*testing.T, *app, *ui.Tester){
		"project": func(t *testing.T, a *app, tt *ui.Tester) {
			if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			click(t, tt, "Jun")
		},
		"perspective": func(t *testing.T, a *app, tt *ui.Tester) { perspectiveMenu(t, tt, "Review") },
		"new perspective": func(t *testing.T, a *app, tt *ui.Tester) {
			perspectiveMenu(t, tt, "New Perspective…")
			tt.Type("Investigation")
			tt.Key(0, ui.KeyEnter)
		},
		"rename perspective": func(t *testing.T, a *app, tt *ui.Tester) {
			perspectiveMenu(t, tt, "Rename Perspective…")
			tt.Key(ui.Super, ui.KeyA)
			tt.Type("Development")
			tt.Key(0, ui.KeyEnter)
		},
		"global": func(t *testing.T, a *app, tt *ui.Tester) { click(t, tt, "Hide Global") },
		"bottom": func(t *testing.T, a *app, tt *ui.Tester) {
			perspectiveMenu(t, tt, "Show Bottom Region")
		},
		"add column": func(t *testing.T, a *app, tt *ui.Tester) {
			menu(t, tt, a, a.state.CurrentPerspective().Columns[0], "Add Column to Right")
		},
		"remove column": func(t *testing.T, a *app, tt *ui.Tester) {
			menu(t, tt, a, a.state.CurrentPerspective().Columns[1], "Remove This Column")
		},
		"new terminal": func(t *testing.T, a *app, tt *ui.Tester) {
			menu(t, tt, a, a.state.CurrentPerspective().Columns[0], "Open View", "New Terminal")
		},
		"close view": func(t *testing.T, a *app, tt *ui.Tester) {
			chooseTabMenu(t, tt, a, a.state.CurrentPerspective().Columns[0].Views[0], "Close Tab")
		},
		"move view": func(t *testing.T, a *app, tt *ui.Tester) {
			chooseTabMenu(t, tt, a, a.state.CurrentPerspective().Columns[0].Views[0], "Move Tab To", "Global")
		},
		"select tab": func(t *testing.T, a *app, tt *ui.Tester) {
			region := a.state.CurrentPerspective().Columns[0]
			view := region.Views[0]
			if _, err := a.state.OpenView(region.ID, layout.ViewTerminal, "Jun"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			click(t, tt, a.title(view))
		},
		"column width": func(t *testing.T, a *app, tt *ui.Tester) {
			divider, ok := tt.Find("Divider")
			if !ok {
				t.Fatal("missing divider")
			}
			tt.ClickAt(divider.X, 100)
			tt.Key(0, ui.KeyRight)
		},
		"bottom height": func(t *testing.T, a *app, tt *ui.Tester) {
			p := a.state.CurrentPerspective()
			if _, err := a.state.RemoveColumn(p.Columns[1].ID); err != nil {
				t.Fatal(err)
			}
			p.BottomVisible = true
			a.state.GlobalVisible = false
			tt.Frame()
			divider, ok := tt.Find("Divider")
			if !ok {
				t.Fatal("missing divider")
			}
			tt.ClickAt(divider.X+20, divider.Y)
			tt.Key(0, ui.KeyDown)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			a, tt := testApp(t)
			path := filepath.Join(t.TempDir(), "config.json")
			a.restoreConfig(path)
			change(t, a, tt)
			tt.Advance(400 * time.Millisecond)
			persistedMatches(t, a, path)
		})
	}
}

func TestDamagedConfigStaysUntouchedWhileEditing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	original := []byte(`{"version":99}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	a := testNewApp(t)
	a.restoreConfig(path)
	tt := ui.NewTester(a.view, 1440, 900)
	if !a.errorOpen {
		t.Fatal("load failure was not presented")
	}
	click(t, tt, "OK")
	a.projectForm.name = "Jun"
	a.projectForm.path = t.TempDir()
	a.submitProject()
	tt.Frame()
	click(t, tt, "Toggle project names")
	tt.Advance(time.Second)
	event := &mygo.CloseEvent{}
	a.beforeClose(event)
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(current, original) || event.DefaultPrevented() {
		t.Fatal("disabled saving overwrote config or blocked quitting")
	}
}

func TestSaveFailureCanCancelQuitAndRetry(t *testing.T) {
	a, tt := testApp(t)
	parent := filepath.Join(t.TempDir(), "blocked")
	path := filepath.Join(parent, "config.json")
	a.restoreConfig(path)
	if err := os.WriteFile(parent, []byte("blocker"), 0600); err != nil {
		t.Fatal(err)
	}
	menu(t, tt, a, a.state.CurrentPerspective().Columns[0], "Open View", "New Terminal")
	tt.Advance(400 * time.Millisecond)
	if !tt.HasText("Layout could not be saved") {
		t.Fatal("save failure not visible")
	}
	click(t, tt, "Continue Editing")
	tt.Advance(time.Second)
	if tt.HasText("Layout could not be saved") {
		t.Fatal("idle frames retried a failed save")
	}
	event := &mygo.CloseEvent{}
	a.beforeClose(event)
	tt.Frame()
	if !event.DefaultPrevented() || !tt.HasText("Quit Without Saving") {
		t.Fatal("failed save did not retain the window with exit choices")
	}
	click(t, tt, "Cancel Quit")
	session := a.sessionFor(a.state.CurrentPerspective().Columns[0].Active)
	session.Terminal.Send([]byte("printf '\\nCANCEL_ALIVE\\n'\n"))
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nCANCEL_ALIVE\n") })
	if err := os.Remove(parent); err != nil {
		t.Fatal(err)
	}
	event = &mygo.CloseEvent{}
	a.beforeClose(event)
	if event.DefaultPrevented() {
		t.Fatal("fixed path still prevented close")
	}
	persistedMatches(t, a, path)
}

func TestRetryAndQuitWithoutSavingActions(t *testing.T) {
	for _, retry := range []bool{true, false} {
		t.Run(map[bool]string{true: "retry", false: "discard"}[retry], func(t *testing.T) {
			a, tt := testApp(t)
			parent := filepath.Join(t.TempDir(), "config")
			path := filepath.Join(parent, "config.json")
			a.restoreConfig(path)
			if err := os.WriteFile(parent, []byte("blocker"), 0600); err != nil {
				t.Fatal(err)
			}
			a.newTerminal(a.state.CurrentPerspective().Columns[0].ID)
			event := &mygo.CloseEvent{}
			a.beforeClose(event)
			tt.Frame()
			if !event.DefaultPrevented() {
				t.Fatal("failed save did not prevent close")
			}
			if retry {
				if err := os.Remove(parent); err != nil {
					t.Fatal(err)
				}
				click(t, tt, "Retry Save")
				persistedMatches(t, a, path)
			} else {
				click(t, tt, "Quit Without Saving")
				event = &mygo.CloseEvent{}
				a.beforeClose(event)
				if event.DefaultPrevented() {
					t.Fatal("explicit discard still blocks close")
				}
				current, err := os.ReadFile(parent)
				if err != nil || string(current) != "blocker" {
					t.Fatal("discard wrote to disk")
				}
			}
		})
	}
}

func TestMissingProjectDirectoryDoesNotInvalidateSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	a := testNewApp(t)
	cwd := filepath.Join(t.TempDir(), "missing")
	if err := a.state.AddProject("Jun", cwd); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(path, a.state); err != nil {
		t.Fatal(err)
	}
	restored := testNewApp(t)
	restored.restoreConfig(path)
	restored.checkCurrentDirectory()
	if restored.state.ActiveProject != "Jun" || !restored.errorOpen || !restored.persistence.enabled {
		t.Fatal("missing directory discarded project or disabled saving")
	}
}

func TestProjectSubmissionIsSavedAndRestored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	a := testNewApp(t)
	a.restoreConfig(path)
	tt := ui.NewTester(a.view, 1440, 900)
	tt.Advance(time.Second)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("startup wrote config: %v", err)
	}
	cwd := t.TempDir()
	a.projectForm.name = "Jun"
	a.projectForm.path = cwd
	a.submitProject()
	tt.Frame()
	tt.Advance(400 * time.Millisecond)
	restored := testNewApp(t)
	restored.restoreConfig(path)
	if restored.state.ActiveProject != "Jun" || restored.state.CurrentProject().CWD != cwd || restored.state.GlobalVisible || len(restored.state.Global.Views) != 0 {
		t.Fatal("project/default layout did not survive restart")
	}
	if len(restored.state.CurrentPerspective().Columns) != 1 || len(restored.state.CurrentPerspective().Columns[0].Views) != 1 {
		t.Fatal("default terminal layout did not survive restart")
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded := ui.NewTester(restored.view, 1440, 900)
	loaded.Advance(time.Second)
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("loading an unchanged snapshot rewrote config")
	}
}
