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
