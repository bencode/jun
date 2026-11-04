package layout

import (
	"reflect"
	"testing"
)

func views(ids ...ViewID) []Location {
	result := make([]Location, len(ids))
	for i, id := range ids {
		result[i] = Location{View: id}
	}
	return result
}

func TestHistoryGoesBackAndForward(t *testing.T) {
	all := func(Location) bool { return true }
	a, b, c := Location{View: 1}, Location{View: 2}, Location{View: 3}
	h := History{}.Navigate(a, all).Navigate(b, all) // at c
	h, to, ok := h.GoBack(c)
	if !ok || to != b || !reflect.DeepEqual(h, History{Before: views(1), After: views(3)}) {
		t.Fatalf("back from c: %v %v %+v", to, ok, h)
	}
	h, to, _ = h.GoBack(b)
	if to != a || len(h.Before) != 0 || !reflect.DeepEqual(h.After, views(3, 2)) {
		t.Fatalf("back from b: %v %+v", to, h)
	}
	if _, _, ok := h.GoBack(a); ok {
		t.Fatal("went back past the first location")
	}
	h, to, _ = h.GoForward(a)
	if to != b || !reflect.DeepEqual(h, History{Before: views(1), After: views(3)}) {
		t.Fatalf("forward from a: %v %+v", to, h)
	}
	// Going somewhere new drops what lay ahead.
	if h = h.Navigate(b, all); !reflect.DeepEqual(h, History{Before: views(1, 2)}) {
		t.Fatalf("navigating kept After: %+v", h)
	}
}

func TestHistoryIgnoresLeavingClosedLocations(t *testing.T) {
	h := History{Before: views(1), After: views(3)}
	if got := h.Navigate(Location{View: 2}, func(Location) bool { return false }); !reflect.DeepEqual(got, h) {
		t.Fatalf("leaving a closed tab changed the history: %+v", got)
	}
}

func TestHistoryLimitDropsTheOldest(t *testing.T) {
	h := History{}
	for i := range historyLimit + 5 {
		h = h.Navigate(Location{View: ViewID(i + 1)}, func(Location) bool { return true })
	}
	if len(h.Before) != historyLimit || h.Before[0].View != 6 {
		t.Fatalf("kept %d locations from %d", len(h.Before), h.Before[0].View)
	}
}

func TestHistoryCleanDropsClosedMergesAndCurrent(t *testing.T) {
	closed := ViewID(9)
	exists := func(l Location) bool { return l.View != closed }
	h := History{Before: views(1, 9, 2, 9, 1, 9, 1, 4), After: views(5, 9, 5, 4)}
	got := h.Clean(Location{View: 4}, exists)
	if !reflect.DeepEqual(got, History{Before: views(1, 2, 1), After: views(5)}) {
		t.Fatalf("clean: %+v", got)
	}
	empty := Location{Region: 7}
	if !SameLocation(empty, Location{Region: 7}) || SameLocation(empty, Location{View: 1, Region: 7}) {
		t.Fatal("an empty region's location should match only that region without a tab")
	}
}

func TestLocationFollowsItsTabAndRevealsIt(t *testing.T) {
	s := New()
	jun := project(t, s, "Jun")
	work, review := jun.Perspectives[0], jun.Perspectives[1]
	terminal := Location{View: work.Columns[1].Active}
	if !s.LocationExists(terminal) {
		t.Fatal("the terminal's location should exist")
	}
	// Moved into Review, the tab is still the location, and revealing it
	// goes there.
	if err := s.MoveView(terminal.View, review.Columns[0].ID, 0); err != nil {
		t.Fatal(err)
	}
	if !s.LocationExists(terminal) {
		t.Fatal("a moved tab's location should exist")
	}
	region, err := s.RevealLocation(terminal)
	if err != nil || region != review.Columns[0].ID || s.CurrentPerspective() != review {
		t.Fatalf("reveal went to region %d in %q: %v", region, s.CurrentPerspective().Name, err)
	}
	if err := s.CloseView(terminal.View); err != nil {
		t.Fatal(err)
	}
	if s.LocationExists(terminal) {
		t.Fatal("a closed tab's location should not exist")
	}
	if s.LocationExists(Location{View: s.Global.Views[0].ID}) {
		t.Fatal("Global's views are no location")
	}

	// An empty region is a location of its own, in its project.
	empty := Location{Region: work.Columns[0].ID}
	if s.LocationExists(empty) {
		t.Fatal("a region with a tab is no empty region")
	}
	if err := s.CloseView(work.Columns[0].Active); err != nil {
		t.Fatal(err)
	}
	if err := s.AddProject("Other", "/other"); err != nil {
		t.Fatal(err)
	}
	work.BottomVisible = false
	if !s.LocationExists(empty) || !s.LocationExists(Location{Region: work.Bottom.ID}) {
		t.Fatal("a column or bottom region should be a location")
	}
	if s.LocationExists(Location{Region: work.Side.ID}) {
		t.Fatal("the side region is no location")
	}
	if region, err := s.RevealLocation(empty); err != nil || region != empty.Region || s.CurrentPerspective() != work {
		t.Fatalf("reveal of the empty region went to %d: %v", region, err)
	}
	if region, _ := s.RevealLocation(Location{Region: work.Bottom.ID}); region != work.Bottom.ID || !work.BottomVisible {
		t.Fatal("revealing the bottom region should show it")
	}
	// Given a tab, the region is no longer empty: there is nothing to go back to.
	if _, err := s.OpenView(empty.Region, ViewTerminal, "Jun"); err != nil {
		t.Fatal(err)
	}
	if s.LocationExists(empty) {
		t.Fatal("an empty region's location should end with its first tab")
	}
}
