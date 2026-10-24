package app

import (
	"fmt"
	"log"
	"slices"

	"github.com/bencode/jun/internal/layout"
)

func (a *app) newTerminal(target layout.RegionID) {
	p := a.state.CurrentProject()
	if p == nil {
		return
	}
	if _, err := a.state.OpenView(target, layout.ViewTerminal, p.Name); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(target)
	r := a.state.Region(target)
	a.report(a.startTerminal(r.Views[len(r.Views)-1]))
}

func (a *app) moveView(view layout.ViewID, target layout.RegionID, before layout.ViewID) {
	err := a.state.MoveView(view, target, before)
	a.report(err)
	if err == nil {
		a.layoutChanged()
		a.focusRegion(target)
	}
}

func (a *app) closeView(region layout.RegionID, view layout.ViewID) {
	r := a.state.Region(region)
	if r == nil || !slices.ContainsFunc(r.Views, func(v *layout.View) bool { return v.ID == view }) {
		a.report(fmt.Errorf("view %d is not in region %d", view, region))
		return
	}
	err := a.state.CloseView(view)
	a.report(err)
	if err == nil {
		a.layoutChanged()
		delete(a.scroll, view)
		delete(a.bindings, view)
		delete(a.terminalErrors, view)
		a.focusRegion(region)
	}
}

func (a *app) report(err error) {
	if err == nil {
		return
	}
	log.Printf("Jun: %v", err)
	a.errorMessage = err.Error()
	a.errorOpen = true
	if a.window != nil {
		a.window.Invalidate()
	}
}

func (a *app) selectProject(name string) {
	previous := a.state.ActiveProject
	if err := a.state.SelectProject(name); err != nil {
		a.report(err)
		return
	}
	if previous != name {
		a.layoutChanged()
	}
	a.focusCurrentPerspective()
	a.checkCurrentDirectory()
}

func (a *app) toggleGlobal() {
	a.state.GlobalVisible = !a.state.GlobalVisible
	if !a.state.GlobalVisible && a.activeRegion == a.state.Global.ID {
		a.focusCurrentPerspective()
	}
	a.layoutChanged()
}

func (a *app) selectPerspective(id layout.PerspectiveID) {
	if p := a.state.CurrentProject(); p != nil && p.ActivePerspective == id {
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
		return
	}
	if err := a.state.SelectPerspective(id); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusCurrentPerspective()
}

func (a *app) addColumn(after layout.RegionID) {
	id, err := a.state.AddColumn(after)
	if err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(id)
}

func (a *app) removeColumn(id layout.RegionID) {
	target, err := a.state.RemoveColumn(id)
	if err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(target)
}

func (a *app) setBottomVisible(bottom layout.RegionID, visible bool) {
	p := a.state.CurrentPerspective()
	if p == nil || p.Bottom.ID != bottom {
		a.report(fmt.Errorf("bottom region not in current perspective: %d", bottom))
		return
	}
	if p.BottomVisible != visible {
		p.BottomVisible = visible
		a.layoutChanged()
	}
	if !visible && a.activeRegion == bottom {
		a.focusCurrentPerspective()
		return
	}
	if r := a.targetRegion(); r != nil {
		a.focusRegion(r.ID)
	}
}
