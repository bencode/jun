package config

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestLoadV1PreservesStateAndSavesV3(t *testing.T) {
	old := legacyFixture()
	// Global's terminal moves to the first column of Jun's current
	// perspective, Review, whose placeholder file view goes.
	want := legacyFixture()
	review := want.Projects[0].Perspectives["review"].Columns[0]
	review.Views, review.Active = want.Global.Views, want.Global.Views[0].ID
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
			previous := want.Projects[i].Perspectives[[]string{"work", "review"}[j]]
			if p.Name != name || p.ID <= 501 || !reflect.DeepEqual(p.Columns, previous.Columns) ||
				!reflect.DeepEqual(p.Bottom, previous.Bottom) || p.BottomVisible != previous.BottomVisible ||
				!reflect.DeepEqual(p.ColumnSplits, previous.ColumnSplits) || p.TopPercent != previous.TopPercent ||
				p.Side == nil || len(p.Side.Views) != 1 || p.SideVisible {
				t.Fatal("migration lost perspective state")
			}
		}
	}
	if s.ActiveProject != "Brain" || s.Projects[0].Perspective(s.Projects[0].ActivePerspective).Name != "Review" ||
		s.CurrentPerspective().Name != "Work" || len(s.Global.Views) != 1 || s.Global.Views[0].Kind != layout.ViewAgents ||
		s.GlobalVisible || !s.RailExpanded || s.CenterPercent != 71 {
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
	if err := json.Unmarshal(data, &doc); err != nil || doc.Version != 3 {
		t.Fatal("next save did not write v3", err)
	}
	restored, err := Load(path)
	if err != nil || !reflect.DeepEqual(restored.Projects, s.Projects) || !reflect.DeepEqual(restored.Global, s.Global) {
		t.Fatal("v3 round trip changed migrated state", err)
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

// v2Document has Global holding terminals of both projects, Agents with a
// layout and Files, and Agents and Files views in Jun's Work perspective.
const v2Document = `{"version":2,"layout":{
"projects":[
 {"name":"Jun","cwd":"/Jun","activePerspective":3,"perspectives":[
  {"id":2,"name":"Work","columns":[{"id":4,"views":[{"id":5,"kind":"terminal","projectName":"Jun"},{"id":6,"kind":"agents","projectName":""}],"activeView":6}],
   "bottom":{"id":7,"views":[{"id":8,"kind":"tree","projectName":"Jun"}],"activeView":8},"bottomVisible":true,"columnSplits":[],"topPercent":75},
  {"id":3,"name":"Review","columns":[{"id":9,"views":[],"activeView":0}],"bottom":{"id":10,"views":[],"activeView":0},"bottomVisible":false,"columnSplits":[],"topPercent":75}]},
 {"name":"Brain","cwd":"/Brain","activePerspective":12,"perspectives":[
  {"id":12,"name":"Work","columns":[{"id":13,"views":[{"id":14,"kind":"terminal","projectName":"Brain"}],"activeView":14}],
   "bottom":{"id":15,"views":[],"activeView":0},"bottomVisible":false,"columnSplits":[],"topPercent":75}]}],
"activeProject":"Jun",
"global":{"id":1,"views":[{"id":16,"kind":"terminal","projectName":"Jun"},{"id":17,"kind":"agents","projectName":"","agentsLayout":"list"},
 {"id":18,"kind":"terminal","projectName":"Brain"},{"id":19,"kind":"tree","projectName":"Jun"}],"activeView":18},
"globalVisible":true,"railExpanded":false,"centerPercent":76}}`

func TestLoadV2MovesGlobalViewsAndAddsSideRegions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(v2Document), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	ids := func(r *layout.Region) []layout.ViewID {
		var result []layout.ViewID
		for _, v := range r.Views {
			result = append(result, v.ID)
		}
		return result
	}
	jun, brain := s.Projects[0], s.Projects[1]
	work, review := jun.Perspectives[0], jun.Perspectives[1]
	cases := []struct {
		name   string
		region *layout.Region
		views  []layout.ViewID
		active layout.ViewID
	}{
		{"Jun Work column", work.Columns[0], []layout.ViewID{5}, 5},
		{"Jun Work bottom", work.Bottom, nil, 0},
		{"Jun Review column", review.Columns[0], []layout.ViewID{16}, 16},
		{"Brain Work column", brain.Perspectives[0].Columns[0], []layout.ViewID{14, 18}, 14},
		{"Global", s.Global, []layout.ViewID{17}, 17},
	}
	for _, c := range cases {
		if got := ids(c.region); !reflect.DeepEqual(got, c.views) || c.region.Active != c.active {
			t.Errorf("%s: views %v, active %d; want %v, %d", c.name, got, c.region.Active, c.views, c.active)
		}
	}
	if s.Global.Views[0].AgentsLayout != layout.AgentsList || !s.GlobalVisible || !work.BottomVisible {
		t.Fatal("migration lost Global's or Work's state")
	}
	for _, p := range []*layout.Perspective{work, review, brain.Perspectives[0]} {
		if p.SideVisible || p.SideWidth != layout.DefaultSideWidth || len(p.Side.Views) != 1 || p.Side.ID <= 19 {
			t.Fatalf("perspective %d: side region %+v", p.ID, p.Side)
		}
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != v2Document {
		t.Fatal("loading rewrote v2", err)
	}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	restored, err := Load(path)
	if err != nil || !reflect.DeepEqual(restored.Projects, s.Projects) || !reflect.DeepEqual(restored.Global, s.Global) {
		t.Fatal("v3 round trip changed migrated state", err)
	}
}

func TestInvalidV2IsNotRepairedOrRewritten(t *testing.T) {
	documents := map[string]string{
		"missing source": strings.Replace(v2Document, `"id":16,"kind":"terminal","projectName":"Jun"`, `"id":16,"kind":"terminal","projectName":"Docs"`, 1),
		"nil column":     strings.Replace(v2Document, `"columns":[{"id":9,"views":[],"activeView":0}]`, `"columns":[null]`, 1),
		"bad active":     strings.Replace(v2Document, `"activeView":8}`, `"activeView":99}`, 1),
		"nil global":     `{"version":2,"layout":{"projects":[],"activeProject":"","global":null,"globalVisible":false,"railExpanded":false,"centerPercent":76}}`,
	}
	for name, document := range documents {
		t.Run(name, func(t *testing.T) {
			if document == v2Document {
				t.Fatal("case did not change the document")
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(document), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatal("accepted invalid v2")
			}
			current, err := os.ReadFile(path)
			if err != nil || string(current) != document {
				t.Fatal("invalid v2 was overwritten", err)
			}
		})
	}
}
