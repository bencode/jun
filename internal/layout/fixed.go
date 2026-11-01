package layout

import "slices"

// DefaultSideWidth is the width, in DIPs, a side region starts with.
const DefaultSideWidth float32 = 260

// Tabs regions (columns and bottom regions) hold views that are opened,
// moved and closed. Fixed regions (Global and the side regions) hold their
// preset's views, in its order.
var (
	tabKinds     = []ViewKind{ViewTerminal, ViewFile, ViewDiff}
	fixedKinds   = []ViewKind{ViewTree, ViewSessions, ViewSkills, ViewAgents}
	sidePreset   = []ViewKind{ViewTree, ViewSessions, ViewSkills}
	globalPreset = []ViewKind{ViewAgents}
)

// Fixed reports whether the region is Global or a side region.
func (s *State) Fixed(id RegionID) bool {
	if s.Global != nil && s.Global.ID == id {
		return true
	}
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			if perspective.Side != nil && perspective.Side.ID == id {
				return true
			}
		}
	}
	return false
}

// fill gives the region its preset's views in order, keeping those it has,
// adding those missing and dropping any others, as a preset that changed
// since the snapshot was saved does.
func (s *State) fill(r *Region, preset []ViewKind, project string) {
	views := make([]*View, 0, len(preset))
	for _, kind := range preset {
		if i := slices.IndexFunc(r.Views, func(v *View) bool { return v.Kind == kind }); i >= 0 {
			views = append(views, r.Views[i])
			continue
		}
		s.nextID++
		views = append(views, &View{ID: ViewID(s.nextID), Kind: kind, ProjectName: project})
	}
	r.Views = views
	if !slices.ContainsFunc(views, func(v *View) bool { return v.ID == r.Active }) {
		r.Active = views[0].ID
	}
}

func (s *State) fillPresets() {
	s.fill(s.Global, globalPreset, "")
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			s.fill(perspective.Side, sidePreset, p.Name)
		}
	}
}

// missingPresetViews counts the views fillPresets adds.
func (s *State) missingPresetViews() uint64 {
	missing := func(r *Region, preset []ViewKind) uint64 {
		n := uint64(0)
		for _, kind := range preset {
			if !slices.ContainsFunc(r.Views, func(v *View) bool { return v.Kind == kind }) {
				n++
			}
		}
		return n
	}
	n := missing(s.Global, globalPreset)
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			n += missing(perspective.Side, sidePreset)
		}
	}
	return n
}
