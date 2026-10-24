package config

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bencode/jun/internal/layout"
)

func legacyFixture() *legacyState {
	region := func(id layout.RegionID, viewID layout.ViewID, kind layout.ViewKind) *layout.Region {
		r := &layout.Region{ID: id, Active: viewID}
		if viewID != 0 {
			r.Views = []*layout.View{{ID: viewID, Kind: kind, ProjectName: "Jun"}}
		}
		return r
	}
	return &legacyState{ActiveProject: "Brain", Global: region(1, 10, layout.ViewTerminal), RailExpanded: true, CenterPercent: 71,
		Projects: []*legacyProject{
			{Name: "Jun", CWD: "/Jun", ActivePerspective: "review", Perspectives: map[string]*legacyPerspective{
				"work": {Columns: []*layout.Region{region(2, 4, layout.ViewTerminal)}, Bottom: region(3, 0, ""), TopPercent: 75},
				"review": {Columns: []*layout.Region{region(5, 6, layout.ViewFile), region(13, 14, layout.ViewDiff)},
					Bottom: region(500, 501, layout.ViewTerminal), ColumnSplits: []float32{39}, TopPercent: 61},
			}},
			{Name: "Brain", CWD: "/Brain", ActivePerspective: "work", Perspectives: map[string]*legacyPerspective{
				"work":   {Columns: []*layout.Region{region(7, 8, layout.ViewTerminal)}, Bottom: region(9, 0, ""), TopPercent: 75},
				"review": {Columns: []*layout.Region{region(11, 0, "")}, Bottom: region(12, 0, ""), TopPercent: 75},
			}},
		}}
}

func legacyDocument(t *testing.T, old *legacyState) []byte {
	t.Helper()
	data, err := json.Marshal(struct {
		Version int          `json:"version"`
		Layout  *legacyState `json:"layout"`
	}{1, old})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLoadV1PreservesStateAndSavesV2(t *testing.T) {
	old := legacyFixture()
	path := filepath.Join(t.TempDir(), "config.json")
	original := legacyDocument(t, old)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range s.Projects {
		previous := old.Projects[i]
		if p.Name != previous.Name || p.CWD != previous.CWD || len(p.Perspectives) != 2 {
			t.Fatal("migration changed a project")
		}
		for j, name := range []string{"Work", "Review"} {
			p := p.Perspectives[j]
			previous := previous.Perspectives[[]string{"work", "review"}[j]]
			if p.Name != name || p.ID <= 501 || !reflect.DeepEqual(p.Columns, previous.Columns) ||
				!reflect.DeepEqual(p.Bottom, previous.Bottom) || p.BottomVisible != previous.BottomVisible ||
				!reflect.DeepEqual(p.ColumnSplits, previous.ColumnSplits) || p.TopPercent != previous.TopPercent {
				t.Fatal("migration lost perspective state")
			}
		}
	}
	if s.ActiveProject != "Brain" || s.Projects[0].Perspective(s.Projects[0].ActivePerspective).Name != "Review" ||
		s.CurrentPerspective().Name != "Work" || !reflect.DeepEqual(s.Global, old.Global) || s.GlobalVisible || !s.RailExpanded || s.CenterPercent != 71 {
		t.Fatal("migration lost selection or global state")
	}
	current, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatal("loading rewrote v1", err)
	}
	id, err := s.AddPerspective("Investigation")
	if err != nil || id <= s.Projects[1].Perspectives[1].ID {
		t.Fatal("allocator did not resume after migrated IDs", err)
	}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil || doc.Version != 2 {
		t.Fatal("next save did not write v2", err)
	}
	restored, err := Load(path)
	if err != nil || !reflect.DeepEqual(restored.Projects, s.Projects) || !reflect.DeepEqual(restored.Global, s.Global) {
		t.Fatal("v2 round trip changed migrated state", err)
	}
}

func TestInvalidV1IsNotRepairedOrRewritten(t *testing.T) {
	cases := map[string]func(*legacyState){
		"missing review": func(s *legacyState) { delete(s.Projects[0].Perspectives, "review") },
		"extra key":      func(s *legacyState) { s.Projects[0].Perspectives["other"] = s.Projects[0].Perspectives["work"] },
		"unknown active": func(s *legacyState) { s.Projects[0].ActivePerspective = "other" },
		"duplicate ID":   func(s *legacyState) { s.Global.ID = 500 },
		"missing source": func(s *legacyState) { s.Global.Views[0].ProjectName = "missing" },
		"nil project":    func(s *legacyState) { s.Projects[0] = nil },
		"nil column":     func(s *legacyState) { s.Projects[0].Perspectives["work"].Columns[0] = nil },
		"ID exhaustion":  func(s *legacyState) { s.Global.ID = layout.RegionID(math.MaxUint64 - 2) },
	}
	for name, modify := range cases {
		t.Run(name, func(t *testing.T) {
			old := legacyFixture()
			modify(old)
			path := filepath.Join(t.TempDir(), "config.json")
			original := legacyDocument(t, old)
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("accepted invalid v1")
			}
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, original) {
				t.Fatal("invalid v1 was overwritten", err)
			}
		})
	}
}

func TestV1RejectsUnknownFields(t *testing.T) {
	data := legacyDocument(t, legacyFixture())
	for _, document := range [][]byte{
		bytes.Replace(data, []byte(`"topPercent":75`), []byte(`"topPercent":75,"id":999`), 1),
		bytes.Replace(data, []byte(`"centerPercent":71`), []byte(`"centerPercent":71,"extra":true`), 1),
	} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, document, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("accepted an unknown legacy field")
		}
	}
}
