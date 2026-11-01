package layout

import (
	"fmt"
	"slices"
	"strings"
)

// newPerspective makes a perspective of the project, its side region
// hidden.
func (s *State) newPerspective(project, name string) *Perspective {
	s.nextID++
	id := PerspectiveID(s.nextID)
	p := &Perspective{ID: id, Name: name, Columns: []*Region{s.newRegion()}, Bottom: s.newRegion(), TopPercent: 75,
		Side: s.newRegion(), SideWidth: DefaultSideWidth}
	s.fill(p.Side, sidePreset, project)
	return p
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
	p := s.newPerspective(project.Name, name)
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

// Views lists the views in the perspective's columns, bottom and side regions.
func (p *Perspective) Views() []*View {
	var views []*View
	for _, r := range append(slices.Clone(p.Columns), p.Bottom, p.Side) {
		views = append(views, r.Views...)
	}
	return views
}

// RemovePerspective removes a perspective of the current project, which
// keeps at least one, returning its views. Its neighbor becomes current.
func (s *State) RemovePerspective(id PerspectiveID) ([]*View, error) {
	project := s.CurrentProject()
	if project == nil || project.Perspective(id) == nil {
		return nil, fmt.Errorf("perspective not in current project: %d", id)
	}
	if len(project.Perspectives) == 1 {
		return nil, fmt.Errorf("a project keeps at least one perspective")
	}
	i := slices.IndexFunc(project.Perspectives, func(p *Perspective) bool { return p.ID == id })
	views := project.Perspectives[i].Views()
	project.Perspectives = slices.Delete(project.Perspectives, i, i+1)
	if project.ActivePerspective == id {
		project.ActivePerspective = project.Perspectives[min(i, len(project.Perspectives)-1)].ID
	}
	return views, nil
}
