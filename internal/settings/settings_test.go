package settings

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadDefaultsChecksAndExpands(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	if s, err := Load(path); err != nil || s.Language != "" || s.Font.Size != 0 {
		t.Fatalf("no file loads as %+v, %v", s, err)
	}
	if s, err := Parse([]byte(Template)); err != nil || s.Theme != "" {
		t.Fatalf("the template loads as %+v, %v", s, err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	s, err := Parse([]byte("language: zh\nfont:\n  size: 15\nclaude:\n  configDirs: [~/.claude, /opt/claude/]\n"))
	if err != nil || s.Language != "zh" || s.Font.Size != 15 ||
		!slices.Equal(s.Claude.ConfigDirs, []string{filepath.Join(home, ".claude"), "/opt/claude"}) {
		t.Fatalf("loaded %+v, %v", s, err)
	}
	for _, bad := range []string{"langauge: zh\n", "language: fr\n", "font:\n  size: 99\n", "claude:\n  configDirs: [rel]\n", "theme: [\n"} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Fatalf("%q was taken", bad)
		}
	}
}

func TestSetChangesOnlyItsKeyAndKeepsComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jun", "settings.yaml")
	if err := Set(path, []string{"font", "size"}, float32(14)); err != nil {
		t.Fatal(err)
	}
	if s, err := Load(path); err != nil || s.Font.Size != 14 {
		t.Fatalf("a new file loads as %+v, %v", s, err)
	}

	if err := os.WriteFile(path, []byte(Template), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, []string{"language"}, "zh"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), Template) || !strings.HasSuffix(string(data), "language: zh\n") {
		t.Fatalf("setting a commented file wrote %q", data)
	}

	written := "# mine\ntheme: Dracula # dark\nclaude:\n  configDirs:\n    - ~/.claude # default\n"
	if err := os.WriteFile(path, []byte(written), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, []string{"theme"}, "Nord"); err != nil {
		t.Fatal(err)
	}
	if err := Set(path, []string{"font", "size"}, float32(16)); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# mine", "theme: Nord # dark", "- ~/.claude # default", "font:\n  size: 16"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("wrote %q, without %q", data, want)
		}
	}
}
