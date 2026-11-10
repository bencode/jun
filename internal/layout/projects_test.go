package layout

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestShelvedProjectIsNeverCurrent(t *testing.T) {
	s := New()
	for _, name := range []string{"A", "B", "C", "D"} {
		if err := s.AddProject(name, "/"+name); err != nil {
			t.Fatal(err)
		}
	}
	shown := func() []string {
		var names []string
		for _, p := range s.ShownProjects() {
			names = append(names, p.Name)
		}
		return names
	}
	shelve := func(name string) {
		t.Helper()
		if err := s.ShelveProject(name); err != nil {
			t.Fatal(err)
		}
	}

	shelve("B")
	if err := s.SelectProject("C"); err != nil {
		t.Fatal(err)
	}
	shelve("C")
	if s.ActiveProject != "D" || !slices.Equal(shown(), []string{"A", "D"}) {
		t.Fatalf("shelving C made %s current, shown %v", s.ActiveProject, shown())
	}
	shelve("D")
	if s.ActiveProject != "A" {
		t.Fatalf("shelving the last project went to %s, not the nearest before it", s.ActiveProject)
	}
	if err := s.ShelveProject("A"); err == nil || s.ActiveProject != "A" || s.CurrentProject().Shelved {
		t.Fatal("shelved the last shown project")
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.ActiveProject = "B" // as in a file edited by hand
	if err := restored.Restore(); err != nil {
		t.Fatal(err)
	}
	if restored.CurrentProject().Shelved || !restored.Projects[2].Shelved {
		t.Fatal("restore lost the shelf or kept the current project on it")
	}

	if _, err := s.RemoveProject("A"); err != nil {
		t.Fatal(err)
	}
	if s.ActiveProject != "B" || s.CurrentProject().Shelved || !slices.Equal(shown(), []string{"B"}) {
		t.Fatalf("removing the last shown project made %s current, shown %v", s.ActiveProject, shown())
	}
	if err := s.SelectProject("D"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(shown(), []string{"B", "D"}) {
		t.Fatalf("selecting D did not put it back in place: %v", shown())
	}
}
