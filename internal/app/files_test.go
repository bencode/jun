package app

import (
	"encoding/json"
	"fmt"
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

func TestFilesSearchShowsMatchesAsATree(t *testing.T) {
	files := make([]string, 600)
	for i := range files {
		files[i] = fmt.Sprintf("gen/f%03d.txt", i)
	}
	if m, more := matchFiles(files, "F0", fileSearchLimit); len(m) != 100 || more {
		t.Fatalf("F0 matched %d, more %v", len(m), more)
	}
	if m, more := matchFiles(files, "gen .txt", fileSearchLimit); len(m) != fileSearchLimit || !more || m[0] != "gen/f000.txt" {
		t.Fatalf("a search past the limit gave %d, more %v", len(m), more)
	}

	a := testNewApp(t)
	dir := gitRepo(t)
	for _, name := range []string{"internal/layout/fixed.go", "internal/layout/fixed_test.go"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package layout\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.state.AddProject("Jun", dir); err != nil {
		t.Fatal(err)
	}
	p := a.state.CurrentPerspective()
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	tt.Key(ui.Super, ui.KeyP)
	tt.Frame()
	if !p.SideVisible || p.Side.Active != p.Side.Views[0].ID {
		t.Fatal("⌘P did not show Files")
	}
	search := &a.trees[p.Side.Active].search
	search.query = "LAYOUT fixe"
	tt.Frame()
	if !tt.HasText("internal") || !tt.HasText("fixed_test.go") || tt.HasText("README.md") {
		t.Fatalf("the results are not the matching files as a tree: %v", tt.Texts())
	}
	click(t, tt, "fixed.go")
	tt.Frame()
	if v := activeView(p.Columns[0]); v == nil || v.File == nil || v.File.Path != "internal/layout/fixed.go" {
		t.Fatal("a click on a result did not show it")
	}
	search.query = "lyfix"
	tt.Frame()
	if !tt.HasText("No files match") {
		t.Fatal("letters out of order matched")
	}
}
