package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSubmitProject(t *testing.T) {
	a := newApp()
	cwd := t.TempDir()
	a.projectForm.open = true
	a.projectForm.name = "  Jun  "
	a.projectForm.path = cwd
	a.submitProject()
	if a.projectForm.open || a.projectForm.error != "" || a.state.ActiveProject != "Jun" || a.state.CurrentProject().CWD != cwd {
		t.Fatalf("failed submission: %+v", a.projectForm)
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
		a := newApp()
		a.projectForm.open = true
		a.projectForm.name = tc.name
		a.projectForm.path = tc.path
		a.submitProject()
		if !a.projectForm.open || a.projectForm.error == "" || len(a.state.Projects) != 0 {
			t.Fatalf("accepted %+v", tc)
		}
	}
}
