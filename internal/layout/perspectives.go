package layout

import (
	"fmt"
	"strings"
)

func (s *State) newPerspective(name string) *Perspective {
	s.nextID++
	id := PerspectiveID(s.nextID)
	return &Perspective{ID: id, Name: name,
		Columns: []*Region{s.newRegion()}, Bottom: s.newRegion(), TopPercent: 75}
}

func (s *State) AddPerspective(name string) (PerspectiveID, error) {
	project := s.CurrentProject()
	if project == nil {
		return 0, fmt.Errorf("no active project")
	}
	name = strings.TrimSpace(name)
	if err := perspectiveNameAvailable(project, 0, name); err != nil {
		return 0, err
	}
	p := s.newPerspective(name)
	project.Perspectives = append(project.Perspectives, p)
	project.ActivePerspective = p.ID
	return p.ID, nil
}

func (s *State) RenamePerspective(id PerspectiveID, name string) error {
	project := s.CurrentProject()
	if project == nil || project.Perspective(id) == nil {
		return fmt.Errorf("perspective not in current project: %d", id)
	}
	name = strings.TrimSpace(name)
	if err := perspectiveNameAvailable(project, id, name); err != nil {
		return err
	}
	project.Perspective(id).Name = name
	return nil
}

func (s *State) SelectPerspective(id PerspectiveID) error {
	project := s.CurrentProject()
	if project == nil || project.Perspective(id) == nil {
		return fmt.Errorf("perspective not in current project: %d", id)
	}
	project.ActivePerspective = id
	return nil
}

func perspectiveNameAvailable(project *Project, except PerspectiveID, name string) error {
	if name == "" {
		return fmt.Errorf("perspective name is required")
	}
	for _, p := range project.Perspectives {
		if p.ID != except && strings.EqualFold(p.Name, name) {
			return fmt.Errorf("perspective name already exists: %s", name)
		}
	}
	return nil
}
