package layout

import (
	"fmt"
	"slices"
)

// Location is where the user is: the active tab of the column or bottom
// region they focused last, or that region while it has no tab. A tab
// names it, so it follows the tab when the tab moves.
type Location struct {
	View   ViewID   // 0 while the region has no tab
	Region RegionID // compared only when View is 0
}

// History is the route history, split by the current location, which it
// does not hold: the layout says where the user is. Before and After each
// end nearest to it.
type History struct {
	Before []Location
	After  []Location
}

const historyLimit = 100 // of Before

// SameLocation reports whether a and b are one place: the same tab, or the
// same region while neither has one.
func SameLocation(a, b Location) bool {
	if a.View != 0 || b.View != 0 {
		return a.View == b.View
	}
	return a.Region == b.Region
}

// tabsOwner finds the column or bottom region with the ID, and its project
// and perspective.
func (s *State) tabsOwner(id RegionID) (*Project, *Perspective, *Region) {
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			for _, r := range append(slices.Clone(perspective.Columns), perspective.Bottom) {
				if r.ID == id {
					return p, perspective, r
				}
			}
		}
	}
	return nil, nil, nil
}

// LocationExists reports whether the location is still in the layout: its
// tab in some column or bottom region, or, without a tab, its region.
func (s *State) LocationExists(l Location) bool {
	if l.View == 0 {
		_, _, r := s.tabsOwner(l.Region)
		return r != nil && len(r.Views) == 0
	}
	r, _ := s.locate(l.View)
	return r != nil && !s.Fixed(r.ID)
}

// RevealLocation makes the location current, as RevealView does for its
// tab, and returns the region to focus.
func (s *State) RevealLocation(l Location) (RegionID, error) {
	if l.View != 0 {
		return s.RevealView(l.View)
	}
	p, perspective, r := s.tabsOwner(l.Region)
	if r == nil {
		return 0, fmt.Errorf("region not found")
	}
	s.enter(p)
	p.ActivePerspective = perspective.ID
	s.reveal(r)
	return r.ID, nil
}

// Navigate records leaving from. A location that no longer exists was
// closed rather than left, so it records nothing and keeps After.
func (h History) Navigate(from Location, exists func(Location) bool) History {
	if !exists(from) {
		return h
	}
	before := append(slices.Clone(h.Before), from)
	return History{Before: before[max(0, len(before)-historyLimit):]}
}

// GoBack returns the history after going back from current, and the
// location to go to; false when there is none.
func (h History) GoBack(current Location) (History, Location, bool) {
	if len(h.Before) == 0 {
		return h, Location{}, false
	}
	to := h.Before[len(h.Before)-1]
	return History{Before: slices.Clone(h.Before[:len(h.Before)-1]), After: append(slices.Clone(h.After), current)}, to, true
}

// GoForward is GoBack the other way.
func (h History) GoForward(current Location) (History, Location, bool) {
	if len(h.After) == 0 {
		return h, Location{}, false
	}
	to := h.After[len(h.After)-1]
	return History{Before: append(slices.Clone(h.Before), current), After: slices.Clone(h.After[:len(h.After)-1])}, to, true
}

// Clean drops the locations that no longer exist, then merges neighbors
// that are one place and drops ends that are current, for every step back
// or forward to go somewhere.
func (h History) Clean(current Location, exists func(Location) bool) History {
	clean := func(side []Location) []Location {
		var result []Location
		for _, l := range side {
			if exists(l) && (len(result) == 0 || !SameLocation(result[len(result)-1], l)) {
				result = append(result, l)
			}
		}
		if len(result) > 0 && SameLocation(result[len(result)-1], current) {
			result = result[:len(result)-1] // merged, so no other end is current
		}
		return result
	}
	return History{Before: clean(h.Before), After: clean(h.After)}
}
