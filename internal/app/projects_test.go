package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSubmitProject(t *testing.T) {
	a := testNewApp(t)
	cwd := t.TempDir()
	a.projectForm.open = true
	a.projectForm.name = "  Jun  "
	a.projectForm.path = cwd
	a.submitProject()
	if a.projectForm.open || a.projectForm.error != "" || a.state.ActiveProject != "Jun" || a.state.CurrentProject().CWD != cwd {
		t.Fatalf("failed submission: %+v", a.projectForm)
	}
	r := a.state.CurrentPerspective().Columns[0]
	if len(r.Views) != 1 || len(a.bindings) != 1 || len(a.terminals.ForProject("Jun")) != 1 || a.sessionFor(r.Active) == nil {
		t.Fatal("project did not start exactly one terminal")
	}
	a.projectForm.open = true
	a.projectForm.name = "jun"
	a.submitProject()
	if !a.projectForm.open || a.projectForm.error == "" || len(a.state.Projects) != 1 {
		t.Fatal("duplicate accepted")
	}
}

func TestFailedProjectSubmissionDoesNotAddProject(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, path string }{{"", root}, {"Jun", file}, {"Jun", filepath.Join(root, "deleted")}}
	for _, tc := range cases {
		a := testNewApp(t)
		a.projectForm.open = true
		a.projectForm.name = tc.name
		a.projectForm.path = tc.path
		a.submitProject()
		if !a.projectForm.open || a.projectForm.error == "" || len(a.state.Projects) != 0 || len(a.bindings) != 0 {
			t.Fatalf("accepted %+v", tc)
		}
	}
}

func TestProjectMenuRenamesAndRemoves(t *testing.T) {
	a, tt := testApp(t)
	work := a.state.CurrentPerspective()
	a.closeView(work.Columns[1].ID, work.Columns[1].Views[0].ID)
	tt.Frame()
	projectMenu := func(name, item string) {
		t.Helper()
		if err := tt.RightClick(name); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem(item); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	projectMenu("Jun", "Rename Project…")
	a.projectRename.name = " Lesscap "
	click(t, tt, "Rename")
	if a.state.ActiveProject != "Lesscap" || len(a.terminals.ForProject("Lesscap")) != 3 || len(a.hiddenTerminals("Lesscap")) != 1 {
		t.Fatal("rename did not follow the project's terminals")
	}
	projectMenu("Lesscap", "Remove Project…")
	click(t, tt, "Cancel")
	if len(a.state.Projects) != 1 || len(a.terminals.All()) != 3 {
		t.Fatal("cancel removed the project")
	}
	projectMenu("Lesscap", "Remove Project…")
	click(t, tt, "Remove")
	if len(a.state.Projects) != 0 || a.state.ActiveProject != "" || len(a.terminals.All()) != 0 || len(a.bindings) != 0 ||
		len(a.state.Global.Views) != 1 || a.state.Validate() != nil {
		t.Fatalf("removal left the project's terminals or views: %v", a.state.Validate())
	}
}

func TestDropDirectoryOnRailOpensProject(t *testing.T) {
	a := testNewApp(t)
	tt := ui.NewTester(a.view, 1440, 900)
	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second, first} {
		if !tt.DropFiles(20, 400, dir) {
			t.Fatal("the rail did not take the directory")
		}
	}
	if len(a.state.Projects) != 2 || a.state.ActiveProject != filepath.Base(first) || len(a.terminals.ForProject(filepath.Base(second))) != 1 {
		t.Fatal("dropping a directory did not open it, or opened an open one again")
	}
	file := filepath.Join(first, "notes.md")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	tt.DropFiles(20, 400, file)
	if len(a.state.Projects) != 2 || !a.errorOpen {
		t.Fatal("a dropped file opened as a project")
	}
}

func TestProjectAbbrevAndShortPath(t *testing.T) {
	for name, want := range map[string]string{
		"jun": "Ju", "jun2": "J2", "youshu": "Yo", "cc-mission-control": "CM", "myApp": "MA", "有数": "有数", "x": "X",
	} {
		if got := projectAbbrev(name); got != want {
			t.Errorf("projectAbbrev(%q) = %q, want %q", name, got, want)
		}
	}
	for cwd, want := range map[string]string{
		"/Users/me": "~", "/Users/me/work/lesscap/jun": "~/work/lesscap/jun", "/Users/me/a/b/c/d": "~/…/c/d",
		"/opt/a/b/c/d": "/…/c/d", "/Users/meta/x": "/Users/meta/x",
	} {
		if got := shortPath(cwd, "/Users/me"); got != want {
			t.Errorf("shortPath(%q) = %q, want %q", cwd, got, want)
		}
	}
}
