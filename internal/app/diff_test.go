package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func TestReviewChangesOpensDiffInCurrentPerspective(t *testing.T) {
	dir := gitRepo(t)
	feature := filepath.Join(t.TempDir(), "feature")
	cmd := exec.Command("git", "-C", dir, "worktree", "add", "-q", "-b", "feature", feature)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(feature, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", dir); err != nil {
		t.Fatal(err)
	}
	if _, err := a.state.AddPerspective("Review"); err != nil {
		t.Fatal(err)
	}
	review := a.state.CurrentPerspective()
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	for range 2 {
		a.reviewChanges()
		tt.Frame()
		click(t, tt, "Review Changes: feature")
		tt.Frame()
	}

	column := review.Columns[0]
	diffs := slices.DeleteFunc(slices.Clone(column.Views), func(v *layout.View) bool { return v.Kind != layout.ViewDiff })
	if a.state.CurrentPerspective() != review || len(diffs) != 1 || column.Active != diffs[0].ID || diffs[0].Diff.Source != "feature" {
		t.Fatalf("Review Changes opened %+v in the current perspective, which is now %q", diffs, a.state.CurrentPerspective().Name)
	}
	tt.Frame()
	st := a.diffs[diffs[0].ID]
	if st == nil || st.base != "main" || len(st.diff.Changes) != 1 || st.diff.Changes[0].Path != "main.go" {
		t.Fatalf("the Diff view read %+v", st)
	}
	if f := st.file; f.state == nil || f.state.Old.String() != "package main\n" || f.state.New.String() != "package main\n\nfunc main() {}\n" {
		t.Fatalf("main.go shows %+v", f)
	}
}
