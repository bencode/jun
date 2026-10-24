package layout

import (
	"fmt"
	"strings"
)

func (s *State) AddProject(name, cwd string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	for _, p := range s.Projects {
		if strings.EqualFold(p.Name, name) {
			return fmt.Errorf("project name already exists: %s", name)
		}
	}
	p := &Project{Name: name, CWD: cwd, ActivePerspective: PerspectiveWork,
		Perspectives: map[PerspectiveKind]*Perspective{}}
	for _, kind := range []PerspectiveKind{PerspectiveWork, PerspectiveReview} {
		perspective := &Perspective{Columns: []*Region{s.newRegion(), s.newRegion()},
			Bottom: s.newRegion(), TopPercent: 75, ColumnSplits: []float32{50}}
		p.Perspectives[kind] = perspective
		if kind == PerspectiveWork {
			for _, r := range perspective.Columns {
				s.appendView(r, ViewTerminal, name)
			}
		} else {
			s.appendView(perspective.Columns[0], ViewTree, name)
			s.appendView(perspective.Columns[1], ViewDiff, name)
			s.appendView(perspective.Columns[1], ViewFile, name)
			perspective.Columns[1].Active = perspective.Columns[1].Views[0].ID
			s.appendView(perspective.Bottom, ViewTerminal, name)
			perspective.BottomVisible = true
		}
	}
	s.Projects = append(s.Projects, p)
	s.ActiveProject = name
	return nil
}

func (s *State) SelectProject(name string) error {
	for _, p := range s.Projects {
		if p.Name == name {
			s.ActiveProject = name
			return nil
		}
	}
	return fmt.Errorf("project not found: %s", name)
}

func (s *State) SelectPerspective(kind PerspectiveKind) error {
	p := s.CurrentProject()
	if p == nil {
		return fmt.Errorf("no active project")
	}
	if p.Perspectives[kind] == nil {
		return fmt.Errorf("unknown perspective: %s", kind)
	}
	p.ActivePerspective = kind
	return nil
}
