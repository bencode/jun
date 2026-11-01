package config

import (
	"fmt"
	"math"
	"slices"

	"github.com/bencode/jun/internal/layout"
)

// migrateV2 gives a version 2 layout what version 3 adds: a hidden side
// region in each perspective, and Global showing only Agents. Global's
// other views move to the first column of their project's current
// perspective; Agents and Files views in columns and bottom regions go.
// Whatever is missing or invalid is left for Restore to report.
func migrateV2(s *layout.State) error {
	if s == nil {
		return fmt.Errorf("layout is missing")
	}
	var perspectives []*layout.Perspective
	for _, p := range s.Projects {
		if p != nil {
			perspectives = append(perspectives, slices.DeleteFunc(slices.Clone(p.Perspectives), isNil)...)
		}
	}
	maximum := maximumID(s)
	if maximum >= math.MaxUint64-uint64(len(perspectives)) {
		return fmt.Errorf("no ID space for side regions")
	}
	for _, p := range perspectives {
		if p.Side == nil {
			maximum++
			p.Side = &layout.Region{ID: layout.RegionID(maximum)}
		}
		if p.SideWidth == 0 {
			p.SideWidth = layout.DefaultSideWidth
		}
		for _, r := range append(slices.Clone(p.Columns), p.Bottom) {
			// Version 1's file views were placeholders, with no file.
			removeViews(r, func(v *layout.View) bool {
				return v.Kind == layout.ViewAgents || v.Kind == layout.ViewTree || v.Kind == layout.ViewFile && v.File == nil
			})
		}
	}
	if s.Global != nil {
		migrateGlobal(s)
	}
	return nil
}

func migrateGlobal(s *layout.State) {
	agents := false
	gone := map[*layout.View]bool{}
	for _, v := range s.Global.Views {
		if v == nil || v.Kind == layout.ViewTree {
			continue
		}
		if v.Kind == layout.ViewAgents {
			gone[v] = agents // only the first stays
			agents = true
			continue
		}
		if column := currentColumn(s, v.ProjectName); column != nil {
			if len(column.Views) == 0 {
				column.Active = v.ID
			}
			column.Views = append(column.Views, v)
			gone[v] = true
		}
	}
	removeViews(s.Global, func(v *layout.View) bool { return v.Kind == layout.ViewTree || gone[v] })
}

// currentColumn finds the first column of the project's current perspective.
func currentColumn(s *layout.State, project string) *layout.Region {
	for _, p := range s.Projects {
		if p == nil || p.Name != project {
			continue
		}
		for _, perspective := range p.Perspectives {
			if perspective != nil && perspective.ID == p.ActivePerspective &&
				len(perspective.Columns) > 0 && perspective.Columns[0] != nil {
				return perspective.Columns[0]
			}
		}
	}
	return nil
}

// removeViews drops the views, selecting the first remaining one when the
// active one goes. An active view that was already missing stays so.
func removeViews(r *layout.Region, drop func(*layout.View) bool) {
	if r == nil {
		return
	}
	active := false
	r.Views = slices.DeleteFunc(r.Views, func(v *layout.View) bool {
		gone := v != nil && drop(v)
		active = active || gone && v.ID == r.Active
		return gone
	})
	if active {
		r.Active = 0
		if len(r.Views) > 0 && r.Views[0] != nil {
			r.Active = r.Views[0].ID
		}
	}
}

func maximumID(s *layout.State) uint64 {
	var maximum uint64
	region := func(r *layout.Region) {
		if r == nil {
			return
		}
		maximum = max(maximum, uint64(r.ID))
		for _, v := range r.Views {
			if v != nil {
				maximum = max(maximum, uint64(v.ID))
			}
		}
	}
	region(s.Global)
	for _, p := range s.Projects {
		if p == nil {
			continue
		}
		for _, perspective := range p.Perspectives {
			if perspective == nil {
				continue
			}
			maximum = max(maximum, uint64(perspective.ID))
			for _, r := range append(slices.Clone(perspective.Columns), perspective.Bottom, perspective.Side) {
				region(r)
			}
		}
	}
	return maximum
}

func isNil(p *layout.Perspective) bool { return p == nil }
