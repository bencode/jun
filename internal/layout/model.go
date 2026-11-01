package layout

// IDs are stable for a view's lifetime, including moves between regions.
type ViewID uint64
type RegionID uint64
type PerspectiveID uint64
type ViewKind string

const (
	ViewTerminal ViewKind = "terminal"
	ViewTree     ViewKind = "tree" // Files, in the side region
	ViewDiff     ViewKind = "diff"
	ViewFile     ViewKind = "file"
	ViewAgents   ViewKind = "agents"   // in Global
	ViewSessions ViewKind = "sessions" // in the side region
	ViewSkills   ViewKind = "skills"   // in the side region
)

// AgentsLayout is how an Agents view shows the agents.
type AgentsLayout string

const (
	AgentsAuto  AgentsLayout = "" // cards when they all fit, else a list
	AgentsCards AgentsLayout = "cards"
	AgentsList  AgentsLayout = "list"
)

type View struct {
	ID           ViewID       `json:"id"`
	Kind         ViewKind     `json:"kind"`
	ProjectName  string       `json:"projectName"`
	AgentsLayout AgentsLayout `json:"agentsLayout,omitempty"`
	File         *FileRef     `json:"file,omitempty"` // a file view's file
}

// FileRef is a file a file view shows: its path in a source of the view's
// project's repository, which Source names: a local branch, or the
// directory of a detached worktree.
type FileRef struct {
	Source string `json:"source"`
	Path   string `json:"path"` // relative to the repository's top level, slash-separated
}
type Region struct {
	ID     RegionID `json:"id"`
	Views  []*View  `json:"views"`
	Active ViewID   `json:"activeView"`
}
type Perspective struct {
	ID            PerspectiveID `json:"id"`
	Name          string        `json:"name"`
	Columns       []*Region     `json:"columns"`
	Bottom        *Region       `json:"bottom"`
	BottomVisible bool          `json:"bottomVisible"`
	ColumnSplits  []float32     `json:"columnSplits"`
	TopPercent    float32       `json:"topPercent"`
	// Side is the perspective's fixed region, left of its columns.
	Side        *Region `json:"side"`
	SideVisible bool    `json:"sideVisible"`
	SideWidth   float32 `json:"sideWidth"` // in DIPs, so that the side keeps it as the room around changes
}
type Project struct {
	Name              string         `json:"name"`
	CWD               string         `json:"cwd"`
	ActivePerspective PerspectiveID  `json:"activePerspective"`
	Perspectives      []*Perspective `json:"perspectives"`
}
type State struct {
	Projects      []*Project `json:"projects"`
	ActiveProject string     `json:"activeProject"`
	Global        *Region    `json:"global"`
	GlobalVisible bool       `json:"globalVisible"`
	RailExpanded  bool       `json:"railExpanded"`
	Chinese       bool       `json:"chinese,omitempty"`
	Theme         string     `json:"theme,omitempty"` // a Ghostty theme's name; Jun's own when empty
	CenterPercent float32    `json:"centerPercent"`
	nextID        uint64
}

func New() *State {
	s := &State{CenterPercent: 76}
	s.Global = s.newRegion()
	s.fill(s.Global, globalPreset, "")
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
		for _, perspective := range p.Perspectives {
			result = append(result, perspective.Columns...)
			result = append(result, perspective.Bottom, perspective.Side)
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
		return p.Perspective(p.ActivePerspective)
	}
	return nil
}

func (p *Project) Perspective(id PerspectiveID) *Perspective {
	for _, perspective := range p.Perspectives {
		if perspective.ID == id {
			return perspective
		}
	}
	return nil
}
