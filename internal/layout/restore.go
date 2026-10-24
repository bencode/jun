package layout

import (
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
)

// Validate checks a snapshot without changing the live ID allocator.
func (s *State) Validate() error {
	_, err := s.validatedMaxID()
	return err
}

// Restore validates a decoded snapshot before resuming its ID allocator.
func (s *State) Restore() error {
	maximum, err := s.validatedMaxID()
	if err != nil {
		return err
	}
	s.nextID = maximum
	return nil
}

type restoreCheck struct {
	projects map[string]bool
	ids      map[uint64]bool
	maximum  uint64
}

func (s *State) validatedMaxID() (uint64, error) {
	if s == nil {
		return 0, fmt.Errorf("layout is missing")
	}
	check := restoreCheck{projects: map[string]bool{}, ids: map[uint64]bool{}}
	if err := check.projectNames(s); err != nil {
		return 0, err
	}
	if !percentage(s.CenterPercent) {
		return 0, fmt.Errorf("centerPercent must be between 0 and 100")
	}
	if err := check.region(s.Global); err != nil {
		return 0, fmt.Errorf("global: %w", err)
	}
	for _, p := range s.Projects {
		if err := check.project(p); err != nil {
			return 0, fmt.Errorf("project %q: %w", p.Name, err)
		}
	}
	return check.maximum, nil
}

func (c *restoreCheck) projectNames(s *State) error {
	for _, p := range s.Projects {
		if p == nil || p.Name == "" || strings.TrimSpace(p.Name) != p.Name {
			return fmt.Errorf("invalid project name")
		}
		// EqualFold matches AddProject, including Unicode case folding.
		for name := range c.projects {
			if strings.EqualFold(name, p.Name) {
				return fmt.Errorf("duplicate project name %q", p.Name)
			}
		}
		if !filepath.IsAbs(p.CWD) {
			return fmt.Errorf("project %q cwd must be absolute", p.Name)
		}
		c.projects[p.Name] = true
	}
	if len(s.Projects) == 0 && s.ActiveProject == "" {
		return nil
	}
	if !c.projects[s.ActiveProject] {
		return fmt.Errorf("activeProject %q does not exist", s.ActiveProject)
	}
	return nil
}

func (c *restoreCheck) project(p *Project) error {
	kinds := []PerspectiveKind{PerspectiveWork, PerspectiveReview}
	if len(p.Perspectives) != len(kinds) || !slices.Contains(kinds, p.ActivePerspective) {
		return fmt.Errorf("invalid perspectives or activePerspective")
	}
	for _, kind := range kinds {
		perspective := p.Perspectives[kind]
		if perspective == nil {
			return fmt.Errorf("perspective %q is missing", kind)
		}
		if err := c.perspective(perspective); err != nil {
			return fmt.Errorf("perspective %q: %w", kind, err)
		}
	}
	return nil
}

func (c *restoreCheck) perspective(p *Perspective) error {
	count := len(p.Columns)
	if count < 1 || count > 3 {
		return fmt.Errorf("column count must be between 1 and 3")
	}
	if len(p.ColumnSplits) != count-1 {
		return fmt.Errorf("columnSplits does not match column count")
	}
	for _, split := range p.ColumnSplits {
		if !percentage(split) {
			return fmt.Errorf("invalid column split percentage")
		}
	}
	if !percentage(p.TopPercent) {
		return fmt.Errorf("invalid topPercent")
	}
	for i, r := range p.Columns {
		if err := c.region(r); err != nil {
			return fmt.Errorf("column %d: %w", i+1, err)
		}
	}
	if err := c.region(p.Bottom); err != nil {
		return fmt.Errorf("bottom: %w", err)
	}
	return nil
}

func (c *restoreCheck) region(r *Region) error {
	if r == nil {
		return fmt.Errorf("region is missing")
	}
	if err := c.identifier(uint64(r.ID)); err != nil {
		return err
	}
	active := len(r.Views) == 0 && r.Active == 0
	for _, v := range r.Views {
		if v == nil {
			return fmt.Errorf("view is missing")
		}
		if err := c.identifier(uint64(v.ID)); err != nil {
			return err
		}
		if !slices.Contains([]ViewKind{ViewTerminal, ViewTree, ViewDiff, ViewFile, ViewAgents}, v.Kind) {
			return fmt.Errorf("unknown view kind %q", v.Kind)
		}
		if v.Kind == ViewAgents {
			if v.ProjectName != "" {
				return fmt.Errorf("Agents view cannot have a source project")
			}
		} else if !c.projects[v.ProjectName] {
			return fmt.Errorf("view %d source project %q does not exist", v.ID, v.ProjectName)
		}
		active = active || v.ID == r.Active
	}
	if !active {
		return fmt.Errorf("region %d activeView is not in its views", r.ID)
	}
	return nil
}

func (c *restoreCheck) identifier(id uint64) error {
	if id == 0 || id == math.MaxUint64 {
		return fmt.Errorf("invalid ID %d", id)
	}
	if c.ids[id] {
		return fmt.Errorf("duplicate ID %d", id)
	}
	c.ids[id] = true
	c.maximum = max(c.maximum, id)
	return nil
}

func percentage(value float32) bool {
	return value > 0 && value < 100
}
