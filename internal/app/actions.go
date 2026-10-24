package app

import (
	"log"

	"github.com/bencode/jun/internal/layout"
)

func (a *app) newTerminal(target layout.RegionID) {
	p := a.state.CurrentProject()
	if p == nil {
		return
	}
	_, err := a.state.OpenView(target, layout.ViewTerminal, p.Name)
	a.report(err)
	if err == nil {
		a.layoutChanged()
		a.pendingFocus = target
	}
}

func (a *app) moveView(view layout.ViewID, target layout.RegionID, before layout.ViewID) {
	err := a.state.MoveView(view, target, before)
	a.report(err)
	if err == nil {
		a.layoutChanged()
		a.pendingFocus = target
	}
}

func (a *app) closeView(region layout.RegionID, view layout.ViewID) {
	err := a.state.CloseView(view)
	a.report(err)
	if err == nil {
		a.layoutChanged()
		delete(a.scroll, view)
		a.pendingFocus = region
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
	a.checkCurrentDirectory()
}

func (a *app) toggleGlobal() {
	a.state.GlobalVisible = !a.state.GlobalVisible
	a.layoutChanged()
}
