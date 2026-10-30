package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/config"
	"github.com/egoist/mygo/ui"
)

func waitTerminal(t *testing.T, tt *ui.Tester, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for terminal")
		}
		time.Sleep(10 * time.Millisecond)
		if tt != nil {
			tt.Frame()
		}
	}
}

func TestTerminalSurvivesCloseReopenAndProjectSwitch(t *testing.T) {
	a, tt := testApp(t)
	r := a.state.CurrentPerspective().Columns[0]
	view := r.Views[0]
	session := a.sessionFor(view.ID)
	a.focusRegion(r.ID)
	tt.Frame()
	tt.Type("printf '\\nLIVE:%s\\n' \"$$\"")
	tt.Key(0, ui.KeyEnter)
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nLIVE:") })
	a.moveView(view.ID, a.state.Global.ID, 0)
	tt.Frame()
	cwdB := t.TempDir()
	if err := a.state.AddProject("Brain", cwdB); err != nil {
		t.Fatal(err)
	}
	a.restoreTerminals()
	a.selectProject("Brain")
	tt.Frame()
	if a.sessionFor(view.ID) != session || session.ProjectName != "Jun" {
		t.Fatal("Global terminal changed ownership")
	}
	a.newTerminal(a.state.Global.ID)
	opened := a.state.Global.Views[len(a.state.Global.Views)-1]
	if s := a.sessionFor(opened.ID); s.CWD != cwdB || s.ProjectName != "Brain" {
		t.Fatal("Global creation used wrong project")
	}
	a.selectProject("Jun")
	a.closeView(a.state.Global.ID, view.ID)
	session.Terminal.Send([]byte("printf '\\nSTILL_ALIVE\\n'\n"))
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nSTILL_ALIVE\n") })
	label := fmt.Sprintf("%s · #%d", a.sessionTitle(session), session.ID)
	menu(t, tt, a, r, "Reopen Terminal", label)
	reopened := r.Views[len(r.Views)-1]
	if reopened.ID == view.ID || a.sessionFor(reopened.ID) != session || !tt.Focused(contentLabel(a, reopened)) {
		t.Fatal("reopen did not reuse the session and focus its content")
	}
	a.selectPerspective(a.state.CurrentProject().Perspectives[1].ID)
	tt.Frame()
	session.Terminal.Send([]byte("printf '\\nREVIEW_ALIVE\\n'\n"))
	waitTerminal(t, tt, func() bool { return strings.Contains(session.Terminal.Text(), "\nREVIEW_ALIVE\n") })
	a.selectPerspective(a.state.CurrentProject().Perspectives[0].ID)
	a.closeView(r.ID, reopened.ID)
	a.reopenTerminal(r.ID, session.ID)
	reopened = r.Views[len(r.Views)-1]
	session.Terminal.Send([]byte("exit 9\n"))
	waitTerminal(t, tt, func() bool { return a.terminals.Get(session.ID) == nil })
	if slices.Contains(r.Views, reopened) {
		t.Fatal("nonzero exit kept the reopened tab")
	}
}

func TestRestoresAllSavedTerminalViewsAsFreshSessions(t *testing.T) {
	a, _ := testApp(t)
	a.newTerminal(a.state.Global.ID)
	if _, err := a.state.AddPerspective("Investigation"); err != nil {
		t.Fatal(err)
	}
	a.newTerminal(a.state.CurrentPerspective().Bottom.ID)
	a.state.CurrentPerspective().BottomVisible = false
	if err := a.state.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	a.restoreTerminals()
	a.state.GlobalVisible = false
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, a.state); err != nil {
		t.Fatal(err)
	}
	restored := testNewApp(t)
	restored.restoreConfig(path)
	restored.restoreTerminals()
	if len(restored.bindings) != len(a.bindings) {
		t.Fatal("hidden terminal views were not restored")
	}
	for view, id := range restored.bindings {
		session := restored.terminals.Get(id)
		if session.Terminal == a.sessionFor(view).Terminal {
			t.Fatal("restoration reused a previous terminal")
		}
		session.Terminal.Send([]byte(fmt.Sprintf("test . -ef %q && printf '\\nRESTORED_OK\\n'\n", session.CWD)))
		waitTerminal(t, nil, func() bool { return strings.Contains(session.Terminal.Text(), "\nRESTORED_OK\n") })
	}
	if restored.state.GlobalVisible || restored.state.ActiveProject != "Brain" {
		t.Fatal("restore changed layout selection")
	}
	before := len(restored.bindings)
	restored.restoreTerminals()
	if len(restored.bindings) != before {
		t.Fatal("repeated restore created duplicate bindings")
	}
}

func TestMissingTerminalDirectoryCanRetryWithoutAddingView(t *testing.T) {
	a := testNewApp(t)
	cwd := filepath.Join(t.TempDir(), "missing")
	if err := a.state.AddProject("Jun", cwd); err != nil {
		t.Fatal(err)
	}
	a.restoreTerminals()
	r := a.state.CurrentPerspective().Columns[0]
	view := r.Views[0]
	if a.sessionFor(view.ID) != nil || a.terminalErrors[view.ID] == nil {
		t.Fatal("missing directory did not preserve failed view")
	}
	tt := ui.NewTester(a.view, 1000, 640)
	if !tt.HasText("Terminal could not start") {
		t.Fatal("failure is not visible")
	}
	if err := os.Mkdir(cwd, 0700); err != nil {
		t.Fatal(err)
	}
	click(t, tt, "Retry")
	if len(r.Views) != 1 || a.sessionFor(view.ID) == nil || a.terminalErrors[view.ID] != nil {
		t.Fatal("retry duplicated view or failed to attach")
	}
	if !tt.Focused(contentLabel(a, view)) {
		t.Fatal("retry did not focus terminal")
	}
}

func TestTerminalInputCompositionClipboardAndResize(t *testing.T) {
	a := testNewApp(t)
	a.projectForm.name, a.projectForm.path = "Jun", t.TempDir()
	a.submitProject()
	tt := ui.NewTester(a.view, 1200, 800)
	view := a.state.CurrentPerspective().Columns[0].Views[0]
	term := a.sessionFor(view.ID).Terminal
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("terminal output: %q", term.Text())
		}
	})
	waitTerminal(t, tt, func() bool { return strings.Contains(term.Text(), "$") })
	active, disabled := tt.TextInputActive()
	if !active || disabled {
		t.Fatal("terminal does not accept input method composition")
	}
	tt.Type("printf '\\n%s\\n' '")
	tt.Compose("zhongwen", 5)
	tt.Type("中文")
	tt.Type("'")
	tt.Key(0, ui.KeyEnter)
	waitTerminal(t, tt, func() bool { return strings.Contains(term.Text(), "\n中文\n") })
	tt.SetClipboard("printf '\\nPASTE_OK\\n'")
	tt.Key(ui.Super, ui.KeyV)
	tt.Key(0, ui.KeyEnter)
	waitTerminal(t, tt, func() bool { return strings.Contains(term.Text(), "\nPASTE_OK\n") })
	tt.Key(ui.Super, ui.KeyA)
	tt.Key(ui.Super, ui.KeyC)
	if !strings.Contains(tt.Clipboard(), "PASTE_OK") {
		t.Fatal("terminal copy did not reach clipboard")
	}
	cols, rows := term.Size()
	tt.SetSize(1000, 640)
	nextCols, nextRows := term.Size()
	if nextCols >= cols || nextRows >= rows {
		t.Fatal("terminal did not resize with view")
	}
}
