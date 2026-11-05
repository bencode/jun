package app

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

// filesApp is an app on a repository, a side bar showing Files, and a
// directory outside it holding a Markdown page, its image and a link's file.
func filesApp(t *testing.T) (*app, *ui.Tester, string, string) {
	t.Helper()
	var dot bytes.Buffer
	if err := png.Encode(&dot, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	for name, data := range map[string][]byte{
		"report.md": []byte("# The Report\n\n![a dot](dot.png)\n\n[next](next.md)\n"),
		"next.md":   []byte("Next page\n"),
		"dot.png":   dot.Bytes(),
	} {
		if err := os.WriteFile(filepath.Join(outside, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a := testNewApp(t)
	repository := gitRepo(t)
	if err := a.state.AddProject("Jun", repository); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	return a, tt, repository, outside
}

func TestFileAtFindsTheWorktreeOrTheDirectory(t *testing.T) {
	a, tt, repository, outside := filesApp(t)
	p := a.state.CurrentPerspective()
	p.Side.Active = p.Side.Views[1].ID
	tt.Frame()
	top, err := filepath.EvalSymlinks(repository)
	if err != nil {
		t.Fatal(err)
	}
	if n := a.fileAt("Jun", filepath.Join(top, "main.go")); n.path != "main.go" || filepath.IsAbs(n.source) {
		t.Fatalf("a file of the worktree: %+v", n)
	}
	if n := a.fileAt("Jun", filepath.Join(outside, "report.md")); n.source != outside || n.path != "report.md" {
		t.Fatalf("a file outside the repository: %+v", n)
	}
}

func TestFilesOutsideTheProjectShowAndRestore(t *testing.T) {
	a, tt, _, outside := filesApp(t)
	column := a.state.CurrentPerspective().Columns[0]
	if err := a.showFile("Jun", filepath.Join(outside, "report.md")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("The Report") {
		t.Fatalf("the page did not show: %v", tt.Texts())
	}
	if err := a.state.Validate(); err != nil {
		t.Fatalf("the tab would not restore: %v", err)
	}
	link, ok := tt.Find("next")
	if !ok {
		t.Fatal("no link")
	}
	tt.ClickAt(link.X+8, link.Y+link.H/2)
	tt.Frame()
	if v := activeView(column); v.File == nil || v.File.Path != "next.md" || !tt.HasText("Next page") {
		t.Fatalf("a link beside the file did not open: %+v", v.File)
	}

	// Opened again, a file is read again.
	again := filepath.Join(outside, "next.md")
	if err := os.WriteFile(again, []byte("Changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.showFile("Jun", again); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("Changed") {
		t.Fatalf("the file was not read again: %v", tt.Texts())
	}

	for _, path := range []string{outside, filepath.Join(outside, "gone.md")} {
		before := len(column.Views)
		if err := a.showFile("Jun", path); err == nil || len(column.Views) != before {
			t.Fatalf("%s opened: %v", path, err)
		}
	}
}

func TestFilesDroppedOnARegionOpenInIt(t *testing.T) {
	a, tt, _, outside := filesApp(t)
	column := a.state.CurrentPerspective().Columns[0]
	shown := func() *layout.View { return activeView(column) }
	if err := a.showFile("Jun", filepath.Join(outside, "next.md")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	before := len(column.Views)
	if !tt.DropFiles(600, 400, filepath.Join(outside, "report.md"), filepath.Join(outside, "dot.png")) {
		t.Fatal("the region did not take the files")
	}
	tt.Frame()
	if len(column.Views) != before+2 || shown().File == nil || shown().File.Path != "dot.png" {
		t.Fatalf("the files did not open in tabs: %d, %+v", len(column.Views), shown().File)
	}
	if !tt.DropFiles(600, 400, outside) {
		t.Fatal("the region did not take the directory")
	}
	tt.Frame()
	if len(column.Views) != before+2 || !a.errorOpen {
		t.Fatal("a directory opened, or was not reported")
	}
}
