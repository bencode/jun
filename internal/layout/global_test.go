package layout

import (
	"math"
	"slices"
	"testing"
)

func near(a, b float32) bool { return math.Abs(float64(a-b)) < 0.001 }

func globalState(t *testing.T) (*State, *View, *View) {
	t.Helper()
	s := New()
	s.GlobalVisible = true
	return s, s.Global.Views[0], s.Global.Views[1]
}

func TestToggleGlobalView(t *testing.T) {
	s, agents, messages := globalState(t)
	if err := s.ToggleGlobalView(agents.ID); err != nil {
		t.Fatal(err)
	}
	if agents.Show || s.Global.Active != messages.ID {
		t.Fatalf("hiding the active view left %+v, active %d", agents, s.Global.Active)
	}

	// The last view shown hides Global, and comes back with it.
	if err := s.ToggleGlobalView(messages.ID); err != nil {
		t.Fatal(err)
	}
	if s.GlobalVisible || !messages.Show {
		t.Fatalf("Global visible %v, messages %+v", s.GlobalVisible, messages)
	}

	// With Global hidden, a view comes with it.
	if err := s.ToggleGlobalView(agents.ID); err != nil {
		t.Fatal(err)
	}
	if !s.GlobalVisible || !agents.Show || !messages.Show {
		t.Fatalf("Global visible %v, %+v, %+v", s.GlobalVisible, agents, messages)
	}
	if s.ToggleGlobalView(999) == nil {
		t.Fatal("toggled a view Global does not have")
	}
}

func TestResizeMovesOnlyTheNeighbors(t *testing.T) {
	views := []*View{{Share: 50}, {Share: 30}, {Share: 20}}
	ResizeGlobalView(views, 60)
	if !near(views[0].Share, 60) || !near(views[1].Share, 20) || views[2].Share != 20 {
		t.Fatalf("shares %v, %v, %v", views[0].Share, views[1].Share, views[2].Share)
	}
	// Past the neighbor, the drag stops, and both keep a share.
	ResizeGlobalView(views, 99)
	if views[1].Share <= 0 || views[2].Share != 20 || !near(views[0].Share+views[1].Share, 80) {
		t.Fatalf("shares %v, %v, %v", views[0].Share, views[1].Share, views[2].Share)
	}
}

func TestFirstShareIsOfTheViewsGiven(t *testing.T) {
	views := []*View{{Share: 50}, {Share: 30}, {Share: 20}}
	if got := FirstShare([]*View{views[0], views[2]}); got < 71.4 || got > 71.5 {
		t.Fatalf("with the second hidden, the first takes %v%%", got)
	}
	if got := FirstShare(views); got != 50 {
		t.Fatalf("shown again, the first takes %v%%", got)
	}
}

func TestReviewsStartHiddenAndKeepTheirChoice(t *testing.T) {
	s := New()
	if err := s.AddProject("web", "/tmp/web"); err != nil {
		t.Fatal(err)
	}
	p := s.CurrentPerspective()
	reviews := p.Side.Views[len(p.Side.Views)-1]
	if reviews.Kind != ViewReviews || reviews.Show {
		t.Fatalf("a new side region's last view: %+v", reviews)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("hidden Reviews: %v", err)
	}
	if err := s.ToggleSideView(reviews.ID); err != nil {
		t.Fatal(err)
	}
	if !reviews.Show || p.Side.Active != reviews.ID {
		t.Fatalf("shown Reviews: %+v, active %d", reviews, p.Side.Active)
	}
	if err := s.ToggleSideView(reviews.ID); err != nil {
		t.Fatal(err)
	}
	if reviews.Show || p.Side.Active != p.Side.Views[0].ID {
		t.Fatalf("hidden Reviews: %+v, active %d", reviews, p.Side.Active)
	}
	p.SideVisible = false
	if err := s.RevealSideView(reviews.ID); err != nil {
		t.Fatal(err)
	}
	if !reviews.Show || !p.SideVisible || p.Side.Active != reviews.ID {
		t.Fatalf("revealed Reviews: %+v, side %v", reviews, p.SideVisible)
	}
}

func TestRestoreHidesReviewsAddedToAnEarlierLayout(t *testing.T) {
	s := New()
	if err := s.AddProject("web", "/tmp/web"); err != nil {
		t.Fatal(err)
	}
	p := s.CurrentPerspective()
	p.Side.Views = p.Side.Views[:2] // as an earlier version saved it
	if err := s.Restore(); err != nil {
		t.Fatal(err)
	}
	reviews := p.Side.Views[2]
	if reviews.Kind != ViewReviews || reviews.Show || p.Side.Active == reviews.ID {
		t.Fatalf("restored side region: %+v, active %d", reviews, p.Side.Active)
	}
}

func TestShowDiffReplacesTheComparisonAndKeepsTheLayout(t *testing.T) {
	s := New()
	if err := s.AddProject("web", "/tmp/web"); err != nil {
		t.Fatal(err)
	}
	r := s.CurrentPerspective().Columns[0]
	first, err := s.OpenDiff(r.ID, "web", DiffRef{Source: "feat", Base: "main", Unified: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.ShowDiff(r.ID, "web", DiffRef{Source: "feat", Base: "dev", Uncommitted: true})
	if err != nil {
		t.Fatal(err)
	}
	v := r.Views[slices.IndexFunc(r.Views, func(v *View) bool { return v.ID == first })]
	if second != first || *v.Diff != (DiffRef{Source: "feat", Base: "dev", Uncommitted: true, Unified: true}) {
		t.Fatalf("ShowDiff = %d, %+v", second, *v.Diff)
	}
}

func TestOpenReviewUsesThePreviewTab(t *testing.T) {
	s := New()
	if err := s.AddProject("web", "/tmp/web"); err != nil {
		t.Fatal(err)
	}
	r := s.CurrentPerspective().Columns[0]
	id, err := s.OpenReview(r.ID, "web", "one", 0)
	if err != nil {
		t.Fatal(err)
	}
	views := len(r.Views)
	if again, err := s.OpenReview(r.ID, "web", "two", id); err != nil || again != id || len(r.Views) != views {
		t.Fatalf("OpenReview = %d, %v; views %d of %d", again, err, len(r.Views), views)
	}
	if _, err := s.OpenReview(r.ID, "web", "../x", 0); err == nil {
		t.Fatal("opened a review whose ID is a path")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
