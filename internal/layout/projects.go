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
	work := s.newPerspective("Work")
	s.appendView(work.Columns[0], ViewTerminal, name)
	p := &Project{Name: name, CWD: cwd, ActivePerspective: work.ID, Perspectives: []*Perspective{work}}

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
