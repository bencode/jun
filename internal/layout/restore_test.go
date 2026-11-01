package layout

import (
	"encoding/json"
	"math"
	"reflect"
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
		"nil region":        func(s *State) { s.CurrentPerspective().Bottom = nil },
		"invalid ratio":     func(s *State) { s.CenterPercent = 100 },
		"invalid splits":    func(s *State) { s.CurrentPerspective().ColumnSplits = []float32{50} },
		"unknown kind":      func(s *State) { s.CurrentPerspective().Columns[0].Views[0].Kind = "other" },
		"agents layout":     func(s *State) { s.CurrentPerspective().Columns[0].Views[0].AgentsLayout = AgentsList },
		"duplicate name":    func(s *State) { p := *s.CurrentProject(); p.Name = "JUN"; s.Projects = append(s.Projects, &p) },
		"files in a column": func(s *State) { s.CurrentPerspective().Columns[0].Views[0].Kind = ViewTree },
		"agents in a column": func(s *State) {
			v := s.CurrentPerspective().Columns[0].Views[0]
			v.Kind, v.ProjectName = ViewAgents, ""
		},
		"terminal in Global": func(s *State) { s.Global.Views[0].Kind, s.Global.Views[0].ProjectName = ViewTerminal, "Jun" },
		"terminal in side":   func(s *State) { s.CurrentPerspective().Side.Views[0].Kind = ViewTerminal },
		"side kind twice":    func(s *State) { s.CurrentPerspective().Side.Views[1].Kind = ViewTree },
		"side project":       func(s *State) { s.CurrentPerspective().Side.Views[0].ProjectName = "" },
		"Global project":     func(s *State) { s.Global.Views[0].ProjectName = "Jun" },
		"nil side":           func(s *State) { s.CurrentPerspective().Side = nil },
		"side width":         func(s *State) { s.CurrentPerspective().SideWidth = 0 },
		"no ID space": func(s *State) {
			side := s.CurrentPerspective().Side
			side.Views, side.Active = side.Views[1:], side.Views[1].ID
			s.CurrentPerspective().Bottom.ID = RegionID(math.MaxUint64 - 1)
		},
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

func TestRestoreGivesFixedRegionsTheirPresets(t *testing.T) {
	s := decodedLayout(t)
	side := s.CurrentPerspective().Side
	skills := side.Views[2]
	// A view a later preset no longer has, selected, in place of Files and Sessions.
	dropped := &View{ID: 900, Kind: ViewAgents, ProjectName: "Jun"}
	side.Views, side.Active = []*View{skills, dropped}, dropped.ID
	s.Global.Views, s.Global.Active = nil, 0
	if err := s.Restore(); err != nil {
		t.Fatal(err)
	}
	var kinds []ViewKind
	for _, v := range side.Views {
		kinds = append(kinds, v.Kind)
		if v.ProjectName != "Jun" || v.Kind != ViewSkills && v.ID <= 900 {
			t.Fatalf("side view %d (%s) of %q", v.ID, v.Kind, v.ProjectName)
		}
	}
	if !reflect.DeepEqual(kinds, sidePreset) || side.Views[2] != skills || side.Active != side.Views[0].ID {
		t.Fatalf("side region restored as %v, active %d", kinds, side.Active)
	}
	if len(s.Global.Views) != 1 || s.Global.Views[0].Kind != ViewAgents || s.Global.Active != s.Global.Views[0].ID || s.Validate() != nil {
		t.Fatal("Global did not get its Agents view")
	}
	id, err := s.OpenView(s.CurrentPerspective().Columns[0].ID, ViewTerminal, "Jun")
	if err != nil || id <= s.Global.Views[0].ID {
		t.Fatalf("new ID %d reuses a preset view's: %v", id, err)
	}
}
