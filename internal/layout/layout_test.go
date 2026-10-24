package layout

import (
	"reflect"
	"slices"
	"testing"
)

func project(t *testing.T, s *State, name string) *Project {
	t.Helper()
	if err := s.AddProject(name, "/"+name); err != nil {
		t.Fatal(err)
	}
	p := s.CurrentPerspective()
	if _, err := s.AddColumn(p.Columns[len(p.Columns)-1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.OpenView(s.CurrentPerspective().Columns[1].ID, ViewTerminal, name); err != nil {
		t.Fatal(err)
	}
	return s.CurrentProject()
}

func TestProjectPerspectiveAndGlobalIsolation(t *testing.T) {
	s := New()
	if s.CurrentProject() != nil || len(s.Global.Views) != 0 {
		t.Fatal("unexpected initial state")
	}
	a := project(t, s, "Jun")
	work := s.CurrentPerspective()
	work.TopPercent = 61
	if err := s.SelectPerspective(PerspectiveReview); err != nil {
		t.Fatal(err)
	}
	review := s.CurrentPerspective()
	review.BottomVisible = false
	global := s.Global
	globalActive := global.Active
	s.GlobalVisible = false
	project(t, s, "Brain")
	if s.Global != global || s.Global.Active != globalActive || s.GlobalVisible {
		t.Fatal("project switch changed Global")
	}
	if err := s.SelectProject(a.Name); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPerspective() != review || review.BottomVisible {
		t.Fatal("review state lost")
	}
	if err := s.SelectPerspective(PerspectiveWork); err != nil {
		t.Fatal(err)
	}
	if s.CurrentPerspective() != work || work.TopPercent != 61 {
		t.Fatal("work state lost")
	}
	for _, name := range []string{"", "  ", "jun", " JUN "} {
		if err := s.AddProject(name, "/other"); err == nil {
			t.Errorf("accepted name %q", name)
		}
	}
	if len(s.Projects) != 2 {
		t.Fatal("invalid add mutated projects")
	}
}

func TestMoveRetainsIdentityAndOriginAcrossProjects(t *testing.T) {
	s := New()
	project(t, s, "Jun")
	p := s.CurrentPerspective()
	source := p.Columns[0]
	view := source.Views[0]
	s.GlobalVisible = false
	if err := s.MoveView(view.ID, s.Global.ID, 0); err != nil {
		t.Fatal(err)
	}
	project(t, s, "Brain")
	if !s.GlobalVisible || s.Global.Active != view.ID || view.ProjectName != "Jun" {
		t.Fatal("Global move lost state")
	}
	destination := s.CurrentPerspective().Columns[0]
	if err := s.MoveView(view.ID, destination.ID, destination.Views[0].ID); err != nil {
		t.Fatal(err)
	}
	if destination.Views[0] != view || view.ProjectName != "Jun" {
		t.Fatal("move rebound source project")
	}
	if len(source.Views) != 0 || source.Active != 0 {
		t.Fatal("source selection was not cleared")
	}
	count := 0
	for _, r := range s.regions() {
		for _, v := range r.Views {
			if v.ID == view.ID {
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("view has %d owners", count)
	}
	bottom := s.CurrentPerspective().Bottom
	if err := s.MoveView(view.ID, bottom.ID, 0); err != nil {
		t.Fatal(err)
	}
	if !s.CurrentPerspective().BottomVisible {
		t.Fatal("move must reveal bottom")
	}
}

func TestReorderAndCloseSelection(t *testing.T) {
	s := New()
	project(t, s, "Jun")
	r := s.CurrentPerspective().Columns[0]
	a := r.Active
	b, err := s.OpenView(r.ID, ViewTerminal, "Jun")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.OpenView(r.ID, ViewTerminal, "Jun")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MoveView(c, r.ID, a); err != nil {
		t.Fatal(err)
	}
	if got := []ViewID{r.Views[0].ID, r.Views[1].ID, r.Views[2].ID}; !reflect.DeepEqual(got, []ViewID{c, a, b}) {
		t.Fatal(got)
	}
	if err := s.CloseView(a); err != nil {
		t.Fatal(err)
	}
	if r.Active != c {
		t.Fatal("closing inactive changed selection")
	}
	if err := s.CloseView(c); err != nil {
		t.Fatal(err)
	}
	if r.Active != b {
		t.Fatal("active close should select right neighbor")
	}
	if err := s.CloseView(b); err != nil {
		t.Fatal(err)
	}
	if len(r.Views) != 0 || r.Active != 0 {
		t.Fatal("last close should leave empty region")
	}
	a, _ = s.OpenView(r.ID, ViewTerminal, "Jun")
	b, _ = s.OpenView(r.ID, ViewTerminal, "Jun")
	if err := s.CloseView(b); err != nil {
		t.Fatal(err)
	}
	if r.Active != a {
		t.Fatal("last tab should select left neighbor")
	}
}

func TestColumnBoundsAndRemovalPreserveViews(t *testing.T) {
	s := New()
	project(t, s, "Jun")
	p := s.CurrentPerspective()
	id, err := s.AddColumn(p.Columns[len(p.Columns)-1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Columns) != 3 || p.ColumnSplits[1] != 50 {
		t.Fatal("three-column sizing")
	}
	if _, err := s.AddColumn(p.Columns[len(p.Columns)-1].ID); err == nil {
		t.Fatal("accepted fourth column")
	}
	view, err := s.OpenView(id, ViewDiff, "Jun")
	if err != nil {
		t.Fatal(err)
	}
	target, err := s.RemoveColumn(id)
	if err != nil {
		t.Fatal(err)
	}
	if target != p.Columns[1].ID || p.Columns[1].Active != view {
		t.Fatal("removal did not preserve the source selection")
	}
	if p.Columns[1].Views[1].ID != view {
		t.Fatal("remove lost views")
	}
	if _, err := s.RemoveColumn(p.Columns[0].ID); err != nil {
		t.Fatal(err)
	}
	if len(p.Columns[0].Views) != 3 {
		t.Fatal("first column removal lost views")
	}
	if _, err := s.RemoveColumn(p.Columns[0].ID); err == nil {
		t.Fatal("removed last column")
	}
}

func TestInvalidOperationsDoNotMutate(t *testing.T) {
	s := New()
	project(t, s, "Jun")
	p := s.CurrentPerspective()
	r := p.Columns[0]
	other := p.Columns[1]
	views := append([]*View(nil), r.Views...)
	active := r.Active
	operations := []func() error{
		func() error { return s.MoveView(active, other.ID, 9999) },
		func() error { return s.MoveView(active, 9999, 0) },
		func() error { return s.MoveView(9999, other.ID, 0) },
		func() error { return s.SelectView(r.ID, other.Active) },
		func() error { return s.CloseView(9999) },
		func() error { _, err := s.RemoveColumn(s.Global.ID); return err },
		func() error { return s.SelectProject("missing") },
		func() error { return s.SelectPerspective("missing") },
		func() error { _, err := s.OpenView(r.ID, ViewTerminal, "missing"); return err },
		func() error { _, err := s.OpenView(r.ID, "missing", "Jun"); return err },
	}
	for i, operation := range operations {
		if operation() == nil {
			t.Fatalf("operation %d accepted invalid input", i)
		}
		if !reflect.DeepEqual(r.Views, views) || r.Active != active || s.ActiveProject != "Jun" || s.CurrentProject().ActivePerspective != PerspectiveWork {
			t.Fatalf("operation %d partially mutated state", i)
		}
	}
}

func TestNewProjectStartsWithOneTerminalView(t *testing.T) {
	s := New()
	if s.GlobalVisible || len(s.Global.Views) != 0 {
		t.Fatal("Global should start empty and hidden")
	}
	if err := s.AddProject("Jun", "/Jun"); err != nil {
		t.Fatal(err)
	}
	work := s.CurrentPerspective()
	review := s.CurrentProject().Perspectives[PerspectiveReview]
	if len(work.Columns) != 1 || len(work.Columns[0].Views) != 1 || work.Columns[0].Views[0].Kind != ViewTerminal || work.BottomVisible || len(work.Bottom.Views) != 0 {
		t.Fatal("Work should contain just one Terminal view")
	}
	if len(review.Columns) != 1 || len(review.Columns[0].Views) != 0 || review.BottomVisible || len(review.Bottom.Views) != 0 {
		t.Fatal("Review should start empty")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAddColumnUsesAndValidatesAnchor(t *testing.T) {
	s := New()
	project(t, s, "A")
	p := s.CurrentPerspective()
	review := s.CurrentProject().Perspectives[PerspectiveReview]
	other := project(t, s, "B").Perspectives[PerspectiveWork].Columns[0].ID
	if err := s.SelectProject("A"); err != nil {
		t.Fatal(err)
	}
	columns, splits, next := slices.Clone(p.Columns), slices.Clone(p.ColumnSplits), s.nextID
	for _, id := range []RegionID{0, s.Global.ID, p.Bottom.ID, review.Columns[0].ID, other} {
		if _, err := s.AddColumn(id); err == nil {
			t.Fatalf("accepted invalid anchor %d", id)
		}
		if !slices.Equal(columns, p.Columns) || !slices.Equal(splits, p.ColumnSplits) || s.nextID != next {
			t.Fatal("invalid anchor mutated layout")
		}
	}
	id, err := s.AddColumn(columns[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Columns[0] != columns[0] || p.Columns[1].ID != id || p.Columns[2] != columns[1] || len(p.Columns[1].Views) != 0 {
		t.Fatal("new empty column did not appear after anchor")
	}
}

func TestRemovingEmptyColumnPreservesDestinationSelection(t *testing.T) {
	for _, first := range []bool{false, true} {
		s := New()
		if err := s.AddProject("A", "/A"); err != nil {
			t.Fatal(err)
		}
		if err := s.SelectPerspective(PerspectiveReview); err != nil {
			t.Fatal(err)
		}
		p := s.CurrentPerspective()
		if _, err := s.AddColumn(p.Columns[0].ID); err != nil {
			t.Fatal(err)
		}
		source, target := p.Columns[1], p.Columns[0]
		if first {
			source, target = target, source
		}
		view, err := s.OpenView(target.ID, ViewFile, "A")
		if err != nil {
			t.Fatal(err)
		}
		destination, err := s.RemoveColumn(source.ID)
		if err != nil {
			t.Fatal(err)
		}
		if destination != target.ID || target.Active != view || len(p.Columns) != 1 {
			t.Fatal("empty removal changed destination selection")
		}
	}
}
