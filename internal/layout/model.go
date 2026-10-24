package layout

// IDs are stable for a view's lifetime, including moves between regions.
type ViewID uint64
type RegionID uint64
type ViewKind string
type PerspectiveKind string

const (
	ViewTerminal      ViewKind        = "terminal"
	ViewTree          ViewKind        = "tree"
	ViewDiff          ViewKind        = "diff"
	ViewFile          ViewKind        = "file"
	ViewAgents        ViewKind        = "agents"
	PerspectiveWork   PerspectiveKind = "work"
	PerspectiveReview PerspectiveKind = "review"
)

type View struct {
	ID          ViewID
	Kind        ViewKind
	ProjectName string
}
type Region struct {
	ID     RegionID
	Views  []*View
	Active ViewID
}
type Perspective struct {
	Columns       []*Region
	Bottom        *Region
	BottomVisible bool
	ColumnSplits  []float32
	TopPercent    float32
}
type Project struct {
	Name              string
	CWD               string
	ActivePerspective PerspectiveKind
	Perspectives      map[PerspectiveKind]*Perspective
}
type State struct {
	Projects      []*Project
	ActiveProject string
	Global        *Region
	GlobalVisible bool
	RailExpanded  bool
	CenterPercent float32
	nextID        uint64
}

func New() *State {
	s := &State{GlobalVisible: true, CenterPercent: 76}
	s.Global = s.newRegion()
	s.appendView(s.Global, ViewAgents, "")
	return s
}

func (s *State) newRegion() *Region {
	s.nextID++
	return &Region{ID: RegionID(s.nextID)}
}

func (s *State) appendView(r *Region, kind ViewKind, project string) ViewID {
	s.nextID++
	v := &View{ID: ViewID(s.nextID), Kind: kind, ProjectName: project}
	r.Views = append(r.Views, v)
	r.Active = v.ID
	return v.ID
}

func (s *State) regions() []*Region {
	result := []*Region{s.Global}
	for _, p := range s.Projects {
		for _, kind := range []PerspectiveKind{PerspectiveWork, PerspectiveReview} {
			perspective := p.Perspectives[kind]
			result = append(result, perspective.Columns...)
			result = append(result, perspective.Bottom)
		}
	}
	return result
}

func (s *State) Region(id RegionID) *Region {
	for _, r := range s.regions() {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (s *State) CurrentProject() *Project {
	for _, p := range s.Projects {
		if p.Name == s.ActiveProject {
			return p
		}
	}
	return nil
}

func (s *State) CurrentPerspective() *Perspective {
	if p := s.CurrentProject(); p != nil {
		return p.Perspectives[p.ActivePerspective]
	}
	return nil
}
