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
	work := s.newPerspective(name, "Work")
	work.SideVisible = true
	s.appendView(work.Columns[0], ViewTerminal, name)
	p := &Project{Name: name, CWD: cwd, ActivePerspective: work.ID, Perspectives: []*Perspective{work}}

	s.Projects = append(s.Projects, p)
	s.ActiveProject = name
	return nil
}

// SelectProject makes the project current, which takes it off the shelf.
func (s *State) SelectProject(name string) error {
	for _, p := range s.Projects {
		if p.Name == name {
			s.ActiveProject = name
			p.Shelved = false
			return nil
		}
	}
	return fmt.Errorf("project not found: %s", name)
}

// ShownProjects are the projects not shelved, in the rail's order.
func (s *State) ShownProjects() []*Project {
	return slices.DeleteFunc(slices.Clone(s.Projects), func(p *Project) bool { return p.Shelved })
}

// ShelveProject shelves the project. The current one hands over to the
// nearest shown project after it, else before it; the last shown project
// cannot be shelved.
func (s *State) ShelveProject(name string) error {
	i := slices.IndexFunc(s.Projects, func(p *Project) bool { return p.Name == name })
	if i < 0 {
		return fmt.Errorf("project not found: %s", name)
	}
	p := s.Projects[i]
	next := s.nearestShown(i, p)
	if next == nil {
		return fmt.Errorf("the last shown project cannot be shelved: %s", name)
	}
	p.Shelved = true
	if s.ActiveProject == name {
		s.ActiveProject = next.Name
	}
	return nil
}

// nearestShown is the first shown project from index i on, else the last
// before it, skipping except.
func (s *State) nearestShown(i int, except *Project) *Project {
	shown := func(p *Project) bool { return !p.Shelved && p != except }
	if j := slices.IndexFunc(s.Projects[i:], shown); j >= 0 {
		return s.Projects[i+j]
	}
	for j := i - 1; j >= 0; j-- {
		if shown(s.Projects[j]) {
			return s.Projects[j]
		}
	}
	return nil
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
// elsewhere, returning every view removed. Its nearest shown neighbor
// becomes current, or with none, its neighbor, taken off the shelf.
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
			next := s.nearestShown(i, nil)
			if next == nil {
				next = s.Projects[min(i, len(s.Projects)-1)]
			}
			s.ActiveProject, next.Shelved = next.Name, false
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
