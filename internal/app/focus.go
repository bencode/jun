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
	if !a.state.Fixed(id) {
		a.lastTabs = id
	}
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
		if p.SideVisible && p.Side.ID == a.activeRegion {
			return p.Side
		}
		return p.Columns[0]
	}
	if a.state.GlobalVisible {
		return a.state.Global
	}
	return nil
}

// tabsTarget is where new tabs go: the active region, or when that is
// Global or the side region, the column or bottom region last focused,
// else the first column.
func (a *app) tabsTarget() *layout.Region {
	r := a.targetRegion()
	if r == nil || !a.state.Fixed(r.ID) {
		return r
	}
	p := a.state.CurrentPerspective()
	if p == nil {
		return nil
	}
	if p.BottomVisible && p.Bottom.ID == a.lastTabs {
		return p.Bottom
	}
	if i := slices.IndexFunc(p.Columns, func(r *layout.Region) bool { return r.ID == a.lastTabs }); i >= 0 {
		return p.Columns[i]
	}
	return p.Columns[0]
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

// visibleRegions orders regions as ⌘[ and ⌘] cycle them: side, columns,
// bottom, Global.
func (a *app) visibleRegions() []*layout.Region {
	var regions []*layout.Region
	if p := a.state.CurrentPerspective(); p != nil {
		if p.SideVisible {
			regions = append(regions, p.Side)
		}
		regions = append(regions, p.Columns...)
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

// neighbor follows the window geometry: the side region sits left of the
// columns and bottom, Global right of them, bottom spans below the columns.
// Directions stop at the edges.
func (a *app) neighbor(d direction) *layout.Region {
	p, r := a.state.CurrentPerspective(), a.targetRegion()
	if p == nil || r == nil {
		return nil
	}
	var global, side *layout.Region
	if a.state.GlobalVisible {
		global = a.state.Global
	}
	if p.SideVisible {
		side = p.Side
	}
	if r == p.Side {
		if d == focusRight {
			return p.Columns[0]
		}
		return nil
	}
	if r == a.state.Global {
		if d == focusLeft {
			return p.Columns[len(p.Columns)-1]
		}
		return nil
	}
	if r == p.Bottom {
		if d == focusLeft {
			return side
		}
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
	if d == focusLeft {
		return side
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
