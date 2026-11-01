package layout

import (
	"fmt"
	"slices"
)

func (s *State) OpenView(target RegionID, kind ViewKind, projectName string) (ViewID, error) {
	r := s.Region(target)
	if r == nil {
		return 0, fmt.Errorf("region not found")
	}
	if s.Fixed(target) {
		return 0, errFixed
	}
	if !slices.Contains(tabKinds, kind) {
		return 0, fmt.Errorf("view kind %s cannot be opened in a tab", kind)
	}
	if !slices.ContainsFunc(s.Projects, func(p *Project) bool { return p.Name == projectName }) {
		return 0, fmt.Errorf("source project not found: %s", projectName)
	}
	id := s.appendView(r, kind, projectName)
	s.reveal(r)
	return id, nil
}

var errFixed = fmt.Errorf("views of Global and side regions cannot be opened, moved or closed")

func (s *State) SelectView(region RegionID, view ViewID) error {
	r := s.Region(region)
	if r == nil || !slices.ContainsFunc(r.Views, func(v *View) bool { return v.ID == view }) {
		return fmt.Errorf("view is not in region")
	}
	r.Active = view
	return nil
}

func (s *State) locate(view ViewID) (*Region, int) {
	for _, r := range s.regions() {
		if i := slices.IndexFunc(r.Views, func(v *View) bool { return v.ID == view }); i >= 0 {
			return r, i
		}
	}
	return nil, -1
}

func (s *State) MoveView(view ViewID, target RegionID, before ViewID) error {
	source, i := s.locate(view)
	destination := s.Region(target)
	if source == nil || destination == nil {
		return fmt.Errorf("view or target region not found")
	}
	if s.Fixed(source.ID) || s.Fixed(destination.ID) {
		return errFixed
	}
	if before != 0 && !slices.ContainsFunc(destination.Views, func(v *View) bool { return v.ID == before }) {
		return fmt.Errorf("destination tab not found")
	}
	if before == view {
		return nil
	}
	v := source.Views[i]
	removeView(source, i)
	index := len(destination.Views)
	if before != 0 {
		index = slices.IndexFunc(destination.Views, func(v *View) bool { return v.ID == before })
	}
	destination.Views = slices.Insert(destination.Views, index, v)
	destination.Active = view
	s.reveal(destination)
	return nil
}

func (s *State) CloseView(view ViewID) error {
	r, i := s.locate(view)
	if r == nil {
		return fmt.Errorf("view not found")
	}
	if s.Fixed(r.ID) {
		return errFixed
	}
	removeView(r, i)
	return nil
}

func removeView(r *Region, i int) {
	active := r.Views[i].ID == r.Active
	r.Views = slices.Delete(r.Views, i, i+1)
	if !active {
		return
	}
	r.Active = 0
	if len(r.Views) > 0 {
		r.Active = r.Views[min(i, len(r.Views)-1)].ID
	}
}

func (s *State) reveal(r *Region) {
	if r == s.Global {
		s.GlobalVisible = true
	}
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			if r == perspective.Bottom {
				perspective.BottomVisible = true
			}
		}
	}
}

// RevealView selects the view in its region and shows the region: the
// region's project and perspective become current, and a hidden bottom
// region or Global appears.
func (s *State) RevealView(view ViewID) (RegionID, error) {
	r, _ := s.locate(view)
	if r == nil {
		return 0, fmt.Errorf("view not found")
	}
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			if r == perspective.Bottom || slices.Contains(perspective.Columns, r) {
				s.ActiveProject, p.ActivePerspective = p.Name, perspective.ID
			}
		}
	}
	r.Active = view
	s.reveal(r)
	return r.ID, nil
}
