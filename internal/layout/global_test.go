package layout

import (
	"math"
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
	if !agents.Hidden || s.Global.Active != messages.ID {
		t.Fatalf("hiding the active view left %+v, active %d", agents, s.Global.Active)
	}

	// The last view shown hides Global, and comes back with it.
	if err := s.ToggleGlobalView(messages.ID); err != nil {
		t.Fatal(err)
	}
	if s.GlobalVisible || messages.Hidden {
		t.Fatalf("Global visible %v, messages %+v", s.GlobalVisible, messages)
	}

	// With Global hidden, a view comes with it.
	if err := s.ToggleGlobalView(agents.ID); err != nil {
		t.Fatal(err)
	}
	if !s.GlobalVisible || agents.Hidden || messages.Hidden {
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
