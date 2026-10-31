package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestCompletions(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"alpha", "Beta", ".hidden", "beam"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "bee"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string][]string{
		root + "/":        {"Beta", "alpha", "beam"},
		root + "/be":      {"Beta", "beam"},
		root + "/.":       {".hidden"},
		root + "/missing": nil,
		root + "/no/such": nil,
	} {
		got, err := completions(query)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		var names []string
		for _, m := range got {
			names = append(names, filepath.Base(m))
		}
		if !slices.Equal(names, want) {
			t.Errorf("completions(%q) = %q, want %q", query, names, want)
		}
	}
	if got := expand("~/work/", "/home/me", "/base"); got != "/home/me/work/" {
		t.Errorf("expand ~ = %q", got)
	}
	if got := expand("../x", "/home/me", "/base"); got != "/base/../x" {
		t.Errorf("expand relative = %q", got)
	}
}

func TestOpenArgs(t *testing.T) {
	a := testNewApp(t)
	wd := t.TempDir()
	if err := os.Mkdir(filepath.Join(wd, "jun"), 0o755); err != nil {
		t.Fatal(err)
	}
	a.openArgs([]string{"-psn_0_1", "jun"}, wd)
	a.openArgs([]string{filepath.Join(wd, "jun")}, "/")
	if len(a.state.Projects) != 1 || a.state.ActiveProject != "jun" || a.state.CurrentProject().CWD != filepath.Join(wd, "jun") {
		t.Fatal("jun <dir> did not open the directory once, relative to its working directory")
	}
	file := filepath.Join(wd, "notes.md")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a.openArgs([]string{file}, wd)
	if len(a.state.Projects) != 1 || !a.errorOpen {
		t.Fatal("jun <file> opened a project, or did not say why not")
	}
}

func TestPaletteOpensDirectory(t *testing.T) {
	a := testNewApp(t)
	tt := ui.NewTester(a.view, 1440, 900)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "jun"), 0o755); err != nil {
		t.Fatal(err)
	}
	a.openPalette(root + "/j")
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	if a.palette.query != filepath.Join(root, "jun")+"/" {
		t.Fatalf("Tab completed to %q", a.palette.query)
	}
	tt.Type("s")
	if a.palette.query != filepath.Join(root, "jun")+"/s" {
		t.Fatalf("typing after Tab gave %q", a.palette.query)
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyEnter)
	if a.palette.open || a.state.ActiveProject != "jun" || a.state.CurrentProject().CWD != filepath.Join(root, "jun") {
		t.Fatal("Enter did not open the directory and close the palette")
	}
}
