package layout

import (
	"fmt"
	"slices"
	"strings"
)

func (s *State) AddProject(name, cwd string) error {
	name = strings.TrimSpace(name)
	if err := projectNameAvailable(s, nil, name); err != nil {
		return err
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

// RenameProject renames the project and every reference to it: views of
// it may have moved into Global or other projects' perspectives.
func (s *State) RenameProject(old, name string) error {
	i := slices.IndexFunc(s.Projects, func(p *Project) bool { return p.Name == old })
	if i < 0 {
		return fmt.Errorf("project not found: %s", old)
	}
	name = strings.TrimSpace(name)
	if err := projectNameAvailable(s, s.Projects[i], name); err != nil {
		return err
	}
	for _, r := range s.regions() {
		for _, v := range r.Views {
			if v.ProjectName == old {
				v.ProjectName = name
			}
		}
	}
	if s.ActiveProject == old {
		s.ActiveProject = name
	}
	s.Projects[i].Name = name
	return nil
}

// RemoveProject removes the project with its perspectives and its views
// elsewhere, returning every view removed. Its neighbor becomes current.
func (s *State) RemoveProject(name string) ([]*View, error) {
	i := slices.IndexFunc(s.Projects, func(p *Project) bool { return p.Name == name })
	if i < 0 {
		return nil, fmt.Errorf("project not found: %s", name)
	}
	var removed []*View
	for _, p := range s.Projects[i].Perspectives {
		removed = append(removed, p.Views()...)
	}
	s.Projects = slices.Delete(s.Projects, i, i+1)
	for _, r := range s.regions() {
		for j := len(r.Views) - 1; j >= 0; j-- {
			if r.Views[j].ProjectName == name {
				removed = append(removed, r.Views[j])
				removeView(r, j)
			}
		}
	}
	if s.ActiveProject == name {
		s.ActiveProject = ""
		if len(s.Projects) > 0 {
			s.ActiveProject = s.Projects[min(i, len(s.Projects)-1)].Name
		}
	}
	return removed, nil
}

func projectNameAvailable(s *State, except *Project, name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	for _, p := range s.Projects {
		if p != except && strings.EqualFold(p.Name, name) {
			return fmt.Errorf("project name already exists: %s", name)
		}
	}
	return nil
}
