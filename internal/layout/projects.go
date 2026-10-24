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
		perspective := &Perspective{Columns: []*Region{s.newRegion()}, Bottom: s.newRegion(), TopPercent: 75}
		p.Perspectives[kind] = perspective
		if kind == PerspectiveWork {
			s.appendView(perspective.Columns[0], ViewTerminal, name)
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
