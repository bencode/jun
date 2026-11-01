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
	if maximum >= math.MaxUint64-s.missingPresetViews() {
		return fmt.Errorf("no ID space for preset views")
	}
	s.nextID = maximum
	s.fillPresets()
	s.dropPlaceholders()
	return nil
}

// dropPlaceholders removes the Diff views of earlier versions, which had
// nothing to compare.
func (s *State) dropPlaceholders() {
	for _, p := range s.Projects {
		for _, perspective := range p.Perspectives {
			for _, r := range append(slices.Clone(perspective.Columns), perspective.Bottom) {
				for i := len(r.Views) - 1; i >= 0; i-- {
					if r.Views[i].Kind == ViewDiff && r.Views[i].Diff == nil {
						removeView(r, i)
					}
				}
			}
		}
	}
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
	if s.TextSize != 0 && (s.TextSize < 8 || s.TextSize > 40) {
		return 0, fmt.Errorf("textSize must be 0, or between 8 and 40")
	}
	if !percentage(s.CenterPercent) {
		return 0, fmt.Errorf("centerPercent must be between 0 and 100")
	}
	if err := check.region(s.Global, fixedKinds, "", true); err != nil {
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
	if len(p.Perspectives) == 0 {
		return fmt.Errorf("invalid perspectives or activePerspective")
	}
	active := false
	for i, perspective := range p.Perspectives {
		if perspective == nil || perspective.Name == "" || strings.TrimSpace(perspective.Name) != perspective.Name {
			return fmt.Errorf("invalid perspective name")
		}
		for _, previous := range p.Perspectives[:i] {
			if strings.EqualFold(previous.Name, perspective.Name) {
				return fmt.Errorf("duplicate perspective name %q", perspective.Name)
			}
		}
		if err := c.perspective(perspective, p.Name); err != nil {
			return fmt.Errorf("perspective %q: %w", perspective.Name, err)
		}
		active = active || perspective.ID == p.ActivePerspective
	}
	if !active {
		return fmt.Errorf("activePerspective %d does not exist", p.ActivePerspective)
	}
	return nil
}

func (c *restoreCheck) perspective(p *Perspective, project string) error {
	if err := c.identifier(uint64(p.ID)); err != nil {
		return err
	}
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
	if !(p.SideWidth > 0 && p.SideWidth <= 10000) {
		return fmt.Errorf("invalid sideWidth")
	}
	for i, r := range p.Columns {
		if err := c.region(r, tabKinds, "", false); err != nil {
			return fmt.Errorf("column %d: %w", i+1, err)
		}
	}
	if err := c.region(p.Bottom, tabKinds, "", false); err != nil {
		return fmt.Errorf("bottom: %w", err)
	}
	if err := c.region(p.Side, fixedKinds, project, true); err != nil {
		return fmt.Errorf("side: %w", err)
	}
	return nil
}

// region checks a region's views. A tabs region holds kinds of any
// project. A fixed region holds each of its kinds once, all of project:
// its own for a side region, none for Global.
func (c *restoreCheck) region(r *Region, kinds []ViewKind, project string, fixed bool) error {
	if r == nil {
		return fmt.Errorf("region is missing")
	}
	if err := c.identifier(uint64(r.ID)); err != nil {
		return err
	}
	active := len(r.Views) == 0 && r.Active == 0
	for i, v := range r.Views {
		if v == nil {
			return fmt.Errorf("view is missing")
		}
		if err := c.identifier(uint64(v.ID)); err != nil {
			return err
		}
		if !slices.Contains(kinds, v.Kind) {
			return fmt.Errorf("view %d: kind %q does not belong in this region", v.ID, v.Kind)
		}
		if v.Kind == ViewAgents {
			if !slices.Contains([]AgentsLayout{AgentsAuto, AgentsCards, AgentsList}, v.AgentsLayout) {
				return fmt.Errorf("unknown agents layout %q", v.AgentsLayout)
			}
		} else if v.AgentsLayout != AgentsAuto {
			return fmt.Errorf("view %d is not an Agents view but has an agents layout", v.ID)
		}
		if (v.Kind == ViewFile) != (v.File != nil) {
			return fmt.Errorf("view %d: only a file view has a file, and it must", v.ID)
		}
		if v.File != nil && (v.File.Source == "" || !validFilePath(v.File.Path)) {
			return fmt.Errorf("view %d: invalid file %q in %q", v.ID, v.File.Path, v.File.Source)
		}
		if (v.Kind == ViewTranscript) != (v.Transcript != nil) {
			return fmt.Errorf("view %d: only a transcript view has a session, and it must", v.ID)
		}
		if v.Transcript != nil {
			if err := validSession(*v.Transcript); err != nil {
				return fmt.Errorf("view %d: %w", v.ID, err)
			}
		}
		// A Diff view with no source is an earlier version's placeholder,
		// which restoring drops.
		if v.Diff != nil && v.Kind != ViewDiff {
			return fmt.Errorf("view %d: only a Diff view compares", v.ID)
		}
		if v.Diff != nil {
			if err := validDiff(*v.Diff); err != nil {
				return fmt.Errorf("view %d: %w", v.ID, err)
			}
		}
		if fixed {
			if v.ProjectName != project {
				return fmt.Errorf("view %d source project %q does not match its region", v.ID, v.ProjectName)
			}
			if slices.ContainsFunc(r.Views[:i], func(previous *View) bool { return previous.Kind == v.Kind }) {
				return fmt.Errorf("view kind %q appears twice", v.Kind)
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
