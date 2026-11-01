package layout

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDynamicPerspectivesKeepIdentityAndLayout(t *testing.T) {
	s := New()
	if err := s.AddProject("Jun", "/Jun"); err != nil {
		t.Fatal(err)
	}
	project, work, global := s.CurrentProject(), s.CurrentPerspective(), s.Global
	reviewID, err := s.AddPerspective("  Review  ")
	if err != nil {
		t.Fatal(err)
	}
	review := s.CurrentPerspective()
	if len(project.Perspectives) != 2 || project.Perspectives[1] != review || review.Name != "Review" ||
		len(review.Columns) != 1 || len(review.Columns[0].Views) != 0 || review.BottomVisible || len(review.Bottom.Views) != 0 {
		t.Fatal("new perspective should be ordered, selected and empty")
	}
	view, err := s.OpenView(review.Columns[0].ID, ViewTerminal, "Jun")
	if err != nil {
		t.Fatal(err)
	}
	review.BottomVisible, review.TopPercent = true, 62
	if err := s.RenamePerspective(reviewID, "  设计审阅  "); err != nil {
		t.Fatal(err)
	}
	if project.Perspective(reviewID) != review || review.Name != "设计审阅" || review.Columns[0].Active != view || review.TopPercent != 62 {
		t.Fatal("rename replaced perspective identity or layout")
	}
	if err := s.SelectPerspective(work.ID); err != nil {
		t.Fatal(err)
	}
	if len(work.Columns[0].Views) != 1 || s.Global != global || s.GlobalVisible {
		t.Fatal("perspective operations changed Work or Global")
	}
	if err := s.SelectPerspective(reviewID); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPerspective() != review || !review.BottomVisible {
		t.Fatal("switching lost perspective state")
	}
	if err := s.AddProject("Brain", "/Brain"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPerspective("设计审阅"); err != nil {
		t.Fatal("names must be scoped to a project", err)
	}
	if err := s.SelectProject("Jun"); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPerspective() != review {
		t.Fatal("project did not remember its selected perspective")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidPerspectiveOperationsDoNotMutate(t *testing.T) {
	s := New()
	if _, err := s.AddPerspective("Work"); err == nil {
		t.Fatal("created a perspective without a project")
	}
	if err := s.AddProject("Jun", "/Jun"); err != nil {
		t.Fatal(err)
	}
	work := s.CurrentPerspective().ID
	if _, err := s.AddPerspective("Σ"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddProject("Brain", "/Brain"); err != nil {
		t.Fatal(err)
	}
	foreign := s.CurrentPerspective().ID
	if err := s.SelectProject("Jun"); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	next := s.nextID
	operations := []func() error{
		func() error { _, err := s.AddPerspective("  "); return err },
		func() error { _, err := s.AddPerspective(" work "); return err },
		func() error { _, err := s.AddPerspective("ς"); return err },
		func() error { return s.RenamePerspective(work, "σ") },
		func() error { return s.RenamePerspective(work, "") },
		func() error { return s.RenamePerspective(foreign, "Other") },
		func() error { return s.SelectPerspective(foreign) },
		func() error { return s.SelectPerspective(0) },
	}
	for _, operation := range operations {
		if err := operation(); err == nil {
			t.Fatal("accepted invalid perspective operation")
		}
		after, err := json.Marshal(s)
		if err != nil || !reflect.DeepEqual(before, after) || s.nextID != next {
			t.Fatal("invalid operation changed state or consumed an ID", err)
		}
	}
}
