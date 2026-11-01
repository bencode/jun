package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
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
	if preview.File.Path != "main.go" || a.previewTabs[column.ID] != preview.ID || fileText(a, preview.ID) != "package main\n" {
		t.Fatal("a click did not show the file in a preview tab")
	}
	click(t, tt, "README.md")
	if v := shown(); v.ID != preview.ID || v.File.Path != "README.md" || fileText(a, v.ID) != "# Jun\n" {
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

// fileText is the text a file view's source render holds.
func fileText(a *app, id layout.ViewID) string {
	for _, render := range a.fileViews[id].renders {
		if s, ok := render.(*sourceRender); ok && s.editor != nil {
			return s.editor.Buffer.String()
		}
	}
	return ""
}

func TestFileViewRendersMarkdownAndImages(t *testing.T) {
	var dot bytes.Buffer
	if err := png.Encode(&dot, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	dir := gitRepo(t)
	files := map[string][]byte{
		"docs/guide.md":    []byte("# The Guide\n\n![a dot](img/dot.png)\n\n[next](next.md)\n"),
		"docs/next.md":     []byte("Next page\n"),
		"docs/img/dot.png": dot.Bytes(),
	}
	for name, data := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", dir); err != nil {
		t.Fatal(err)
	}
	p := a.state.CurrentPerspective()
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	search := &a.trees[p.Side.Active].search
	open := func(query, name string) *layout.View {
		t.Helper()
		search.query = query
		tt.Frame()
		click(t, tt, name)
		tt.Frame()
		return activeView(p.Columns[0])
	}

	guide := open("guide", "guide.md")
	if !tt.HasText("The Guide") || tt.HasText("# The Guide") || fileText(a, guide.ID) == "" {
		t.Fatal("Markdown did not show as a page to read")
	}
	click(t, tt, "Source")
	tt.Frame()
	if a.fileViews[guide.ID].shown != 1 || tt.HasText("The Guide") {
		t.Fatal("the switch did not show the source")
	}
	click(t, tt, "Reading")
	tt.Frame()
	// The paragraph, as wide as the page, shows "next" too: click its start.
	link, ok := tt.Find("next")
	if !ok {
		t.Fatal("no link")
	}
	tt.ClickAt(link.X+8, link.Y+link.H/2)
	tt.Frame()
	if v := activeView(p.Columns[0]); v.File == nil || v.File.Path != "docs/next.md" {
		t.Fatal("a link to a file of the repository did not open it")
	}
	open("dot", "dot.png")
	if !tt.HasText("2 × 2") {
		t.Fatalf("the image did not show with its size: %v", tt.Texts())
	}
}

func TestTextSizeZoomsEveryFileTab(t *testing.T) {
	dir := gitRepo(t)
	for _, name := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", dir); err != nil {
		t.Fatal(err)
	}
	p := a.state.CurrentPerspective()
	if _, err := a.state.AddColumn(p.Columns[0].ID); err != nil {
		t.Fatal(err)
	}
	ids := make([]layout.ViewID, 2)
	for i, name := range []string{"a.go", "b.go"} {
		id, err := a.state.OpenFile(p.Columns[i].ID, "Jun", layout.FileRef{Source: "main", Path: name}, 0)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	sizes := func() [2]float32 {
		t.Helper()
		tt.Frame()
		var got [2]float32
		for i, id := range ids {
			for _, render := range a.fileViews[id].renders {
				if s, ok := render.(*sourceRender); ok && s.editor != nil {
					got[i] = s.editor.Font.Size
				}
			}
		}
		return got
	}

	tt.Key(ui.Super, ui.KeyEqual)
	tt.Key(ui.Super|ui.Shift, ui.KeyEqual)
	if got := sizes(); got != [2]float32{15, 15} {
		t.Fatalf("⌘+ twice gave sizes %v", got)
	}
	tt.Key(ui.Super, ui.Key0)
	if got := sizes(); got != [2]float32{13, 13} {
		t.Fatalf("⌘0 gave sizes %v", got)
	}
	a.setTextSize(100)
	tt.Key(ui.Super, ui.KeyMinus)
	if got := sizes(); got != [2]float32{39, 39} {
		t.Fatalf("⌘− from the largest size gave sizes %v", got)
	}

	data, err := json.Marshal(a.state)
	if err != nil {
		t.Fatal(err)
	}
	restored := &layout.State{}
	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatal(err)
	}
	if err := restored.Restore(); err != nil || restored.TextSize != 39 {
		t.Fatalf("the text size did not survive saving: %v, %v", restored.TextSize, err)
	}
}
