package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bencode/jun/internal/layout"
)

func snapshot(t *testing.T) *layout.State {
	t.Helper()
	s := layout.New()
	if err := s.AddProject("Jun", filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatal(err)
	}
	view := s.CurrentPerspective().Columns[0].Active
	if err := s.AddProject("Brain", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	// A view keeps its source project in another project's perspective.
	if err := s.MoveView(view, s.CurrentPerspective().Columns[0].ID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPerspective("Review"); err != nil {
		t.Fatal(err)
	}
	p := s.CurrentPerspective()
	if _, err := s.AddColumn(p.Columns[0].ID); err != nil {
		t.Fatal(err)
	}
	p.TopPercent = 62
	p.ColumnSplits[0] = 39
	p.BottomVisible = true
	p.SideVisible, p.SideWidth, p.Side.Active = true, 300, p.Side.Views[0].ID
	if _, err := s.OpenView(p.Bottom.ID, layout.ViewTerminal, "Jun"); err != nil {
		t.Fatal(err)
	}
	s.GlobalVisible = false
	s.RailExpanded = true
	s.CenterPercent = 71
	return s
}

func TestSnapshotRoundTripPreservesSourceAndLayout(t *testing.T) {
	s := snapshot(t)
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	// The unexported allocator is reconstructed from live IDs, not serialized.
	if !reflect.DeepEqual(s.Projects, restored.Projects) || !reflect.DeepEqual(s.Global, restored.Global) || restored.ActiveProject != s.ActiveProject || restored.GlobalVisible || !restored.RailExpanded || restored.CenterPercent != 71 {
		t.Fatal("snapshot lost project, global, or layout state")
	}
	moved := restored.Projects[1].Perspectives[0].Columns[0]
	if moved.Views[len(moved.Views)-1].ProjectName != "Jun" || restored.ActiveProject != "Brain" {
		t.Fatal("source project was rebound")
	}
	id, err := restored.OpenView(restored.CurrentPerspective().Columns[0].ID, layout.ViewTerminal, "Jun")
	if err != nil || id <= restored.CurrentPerspective().Bottom.Active {
		t.Fatalf("restored IDs conflict: %d: %v", id, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("config mode: %v %v", info, err)
	}
}

func TestLoadRejectsBrokenAndUnsupportedDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if _, err := Load(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing config: %v", err)
	}
	for _, data := range []string{`{`, `{"version":99}`, `{"version":1}`, `{"version":2}`, `{"version":2,"layout":null}`, `{"version":1,"layout":{},"extra":true}`, `{"version":1} {}`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}

func TestSaveFailurePreservesPriorConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s := snapshot(t)
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Global.Active = 999
	if err := Save(path, s); err == nil {
		t.Fatal("saved invalid state")
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, current) {
		t.Fatal("invalid save overwrote existing config")
	}
	s.Global.Active = s.Global.Views[0].ID
	if err := Save(filepath.Join(path, "child.json"), s); err == nil {
		t.Fatal("missing filesystem error")
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".config-*.tmp"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("left temporary files: %v, %v", matches, err)
	}
}

func TestSavingDoesNotReuseClosedViewIDs(t *testing.T) {
	s := snapshot(t)
	region := s.CurrentPerspective().Columns[0].ID
	removed, err := s.OpenView(region, layout.ViewTerminal, "Jun")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CloseView(removed); err != nil {
		t.Fatal(err)
	}
	if err := Save(filepath.Join(t.TempDir(), "config.json"), s); err != nil {
		t.Fatal(err)
	}
	next, err := s.OpenView(region, layout.ViewTerminal, "Jun")
	if err != nil || next <= removed {
		t.Fatalf("save rewound allocator: %d after %d: %v", next, removed, err)
	}
}
