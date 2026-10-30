package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
)

type direction int

const (
	focusLeft direction = iota
	focusRight
	focusUp
	focusDown
)

func (a *app) focusRegion(id layout.RegionID) {
	a.activeRegion = id
	a.pendingFocus = id
}

func (a *app) targetRegion() *layout.Region {
	if a.state.GlobalVisible && a.activeRegion == a.state.Global.ID {
		return a.state.Global
	}
	if p := a.state.CurrentPerspective(); p != nil {
		for _, r := range p.Columns {
			if r.ID == a.activeRegion {
				return r
			}
		}
		if p.BottomVisible && p.Bottom.ID == a.activeRegion {
			return p.Bottom
		}
		return p.Columns[0]
	}
	if a.state.GlobalVisible {
		return a.state.Global
	}
	return nil
}

func (a *app) focusCurrentPerspective() {
	if p := a.state.CurrentPerspective(); p != nil {
		a.focusRegion(p.Columns[0].ID)
	}
}

func (a *app) cycleTab(delta int) {
	r := a.targetRegion()
	if r == nil || len(r.Views) == 0 {
		return
	}
	index := slices.IndexFunc(r.Views, func(v *layout.View) bool { return v.ID == r.Active })
	a.selectTab(r, r.Views[(index+delta+len(r.Views))%len(r.Views)].ID)
}

func (a *app) selectTab(r *layout.Region, view layout.ViewID) {
	if err := a.state.SelectView(r.ID, view); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(r.ID)
}

// visibleRegions orders regions as ⌘[ and ⌘] cycle them: columns, bottom, Global.
func (a *app) visibleRegions() []*layout.Region {
	var regions []*layout.Region
	if p := a.state.CurrentPerspective(); p != nil {
		regions = slices.Clone(p.Columns)
		if p.BottomVisible {
			regions = append(regions, p.Bottom)
		}
	}
	if a.state.GlobalVisible {
		regions = append(regions, a.state.Global)
	}
	return regions
}

func (a *app) cycleRegion(delta int) {
	regions := a.visibleRegions()
	if len(regions) == 0 {
		return
	}
	index := slices.Index(regions, a.targetRegion())
	a.focusRegion(regions[(index+delta+len(regions))%len(regions)].ID)
}

func (a *app) focusDirection(d direction) {
	if target := a.neighbor(d); target != nil {
		a.focusRegion(target.ID)
	}
}

// neighbor follows the window geometry: Global sits right of the columns,
// bottom spans below them. Directions stop at the edges.
func (a *app) neighbor(d direction) *layout.Region {
	p, r := a.state.CurrentPerspective(), a.targetRegion()
	if p == nil || r == nil {
		return nil
	}
	var global *layout.Region
	if a.state.GlobalVisible {
		global = a.state.Global
	}
	if r == a.state.Global {
		if d == focusLeft {
			return p.Columns[len(p.Columns)-1]
		}
		return nil
	}
	if r == p.Bottom {
		if d == focusRight {
			return global
		}
		if d == focusUp {
			return p.Columns[0]
		}
		return nil
	}
	index := slices.Index(p.Columns, r)
	if d == focusLeft && index > 0 {
		return p.Columns[index-1]
	}
	if d == focusRight && index < len(p.Columns)-1 {
		return p.Columns[index+1]
	}
	if d == focusRight {
		return global
	}
	if d == focusDown && p.BottomVisible {
		return p.Bottom
	}
	return nil
}
