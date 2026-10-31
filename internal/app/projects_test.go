package app

import (
	"os"
	"path/filepath"
	"testing"
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
