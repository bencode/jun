package layout

import (
	"encoding/json"
	"testing"
)

func decodedLayout(t *testing.T) *State {
	t.Helper()
	original := New()
	if err := original.AddProject("Jun", "/Jun"); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded State
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	return &decoded
}

func TestRestoreContinuesIDsAcrossHiddenRegions(t *testing.T) {
	s := decodedLayout(t)
	hidden := s.CurrentPerspective().Bottom
	hidden.ID = 500
	if err := s.Restore(); err != nil {
		t.Fatal(err)
	}
	id, err := s.OpenView(s.CurrentPerspective().Columns[0].ID, ViewTerminal, "Jun")
	if err != nil || id <= 500 {
		t.Fatalf("new ID %d, error %v", id, err)
	}
}

func TestRestoreRejectsInvalidReferencesAndShapes(t *testing.T) {
	cases := map[string]func(*State){
		"duplicate owner": func(s *State) {
			s.Global.Views = s.CurrentPerspective().Columns[0].Views
			s.Global.Active = s.Global.Views[0].ID
		},
		"missing source":      func(s *State) { s.CurrentPerspective().Columns[0].Views[0].ProjectName = "missing" },
		"missing active":      func(s *State) { s.CurrentPerspective().Columns[0].Active = 999 },
		"empty region active": func(s *State) { s.Global.Active = 999 },
		"missing perspective": func(s *State) { s.CurrentProject().ActivePerspective = 999 },
		"no perspectives":     func(s *State) { s.CurrentProject().Perspectives = nil },
		"nil perspective":     func(s *State) { s.CurrentProject().Perspectives[0] = nil },
		"perspective ID":      func(s *State) { s.CurrentPerspective().ID = PerspectiveID(s.Global.ID) },
		"blank perspective":   func(s *State) { s.CurrentPerspective().Name = "  " },
		"padded perspective":  func(s *State) { s.CurrentPerspective().Name = " Work " },
		"duplicate perspective": func(s *State) {
			p := *s.CurrentPerspective()
			p.ID, p.Name = 999, "work"
			s.CurrentProject().Perspectives = append(s.CurrentProject().Perspectives, &p)
		},
		"nil region":     func(s *State) { s.CurrentPerspective().Bottom = nil },
		"invalid ratio":  func(s *State) { s.CenterPercent = 100 },
		"invalid splits": func(s *State) { s.CurrentPerspective().ColumnSplits = []float32{50} },
		"unknown kind":   func(s *State) { s.CurrentPerspective().Columns[0].Views[0].Kind = "other" },
		"duplicate name": func(s *State) { p := *s.CurrentProject(); p.Name = "JUN"; s.Projects = append(s.Projects, &p) },
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			s := decodedLayout(t)
			modify(s)
			if err := s.Restore(); err == nil {
				t.Fatal("accepted invalid layout")
			}
			if s.nextID != 0 {
				t.Fatal("failed restoration mutated allocator")
			}
		})
	}
}

func TestValidateDoesNotRewindLiveAllocator(t *testing.T) {
	s := New()
	if err := s.AddProject("Jun", "/Jun"); err != nil {
		t.Fatal(err)
	}
	region := s.CurrentPerspective().Columns[0].ID
	removed, err := s.OpenView(region, ViewTerminal, "Jun")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CloseView(removed); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	next, err := s.OpenView(region, ViewTerminal, "Jun")
	if err != nil || next <= removed {
		t.Fatalf("ID reused: %d after %d: %v", next, removed, err)
	}
}
