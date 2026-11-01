package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

// gitRepo makes a repository on main with a committed Go file and a
// README not yet committed.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, text string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	write("main.go", "package main\n")
	run("add", ".")
	run("commit", "-q", "-m", "one")
	write("README.md", "# Jun\n")
	return dir
}

func TestFilesOpensPreviewAndPinnedTabs(t *testing.T) {
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", gitRepo(t)); err != nil {
		t.Fatal(err)
	}
	p := a.state.CurrentPerspective()
	p.SideVisible, p.Side.Active = true, p.Side.Views[0].ID
	column := p.Columns[0]
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	shown := func() *layout.View {
		t.Helper()
		tt.Frame()
		v := activeView(column)
		if v == nil || v.File == nil {
			t.Fatal("no file view in the column")
		}
		return v
	}

	click(t, tt, "main.go")
	preview := shown()
	if preview.File.Path != "main.go" || a.previewTabs[column.ID] != preview.ID || a.fileViews[preview.ID].editor.Buffer.String() != "package main\n" {
		t.Fatal("a click did not show the file in a preview tab")
	}
	click(t, tt, "README.md")
	if v := shown(); v.ID != preview.ID || v.File.Path != "README.md" || a.fileViews[v.ID].editor.Buffer.String() != "# Jun\n" {
		t.Fatal("the next click did not replace the preview with an uncommitted file")
	}
	tt.Key(0, ui.KeyEnter)
	if len(column.Views) != 2 || a.previewTabs[column.ID] != 0 {
		t.Fatal("Enter did not keep the preview as a tab of its own")
	}

	data, err := json.Marshal(a.state)
	if err != nil {
		t.Fatal(err)
	}
	restored := &layout.State{}
	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatal(err)
	}
	if err := restored.Restore(); err != nil {
		t.Fatal(err)
	}
	if v := activeView(restored.CurrentPerspective().Columns[0]); v == nil || *v.File != (layout.FileRef{Source: "main", Path: "README.md"}) {
		t.Fatal("the file tab did not survive saving")
	}
}
