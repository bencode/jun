package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
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
	index = (index + delta + len(r.Views)) % len(r.Views)
	if err := a.state.SelectView(r.ID, r.Views[index].ID); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(r.ID)
}
