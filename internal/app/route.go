package app

import "github.com/bencode/jun/internal/layout"

// routeState is the route history and where the user was when it last
// recorded, which the next move leaves from.
type routeState struct {
	history layout.History
	mark    layout.Location
}

// location is where the user is: where new tabs go, so the side region and
// Global, which take the focus but no tabs, leave it.
func (a *app) location() layout.Location {
	r := a.tabsTarget()
	if r == nil {
		return layout.Location{}
	}
	return layout.Location{View: r.Active, Region: r.ID}
}

// sampleRoute records a move from where the user was. The view calls it
// as it ends, once the build has brought the focused region in line with
// the keyboard focus that handlers and clicks moved: an action is one
// move, however many steps it took.
func (a *app) sampleRoute() {
	current := a.location()
	if !layout.SameLocation(a.route.mark, current) {
		a.route.history = a.route.history.Navigate(a.route.mark, a.state.LocationExists)
	}
	a.route.history = a.route.history.Clean(current, a.state.LocationExists)
	a.route.mark = current
}

// quietly makes a change of location that is no move, such as going back
// or closing a tab: it records any move before it, then takes where the
// change ends as where the user is.
func (a *app) quietly(change func()) {
	a.sampleRoute()
	change()
	a.route.mark = a.location()
}

func (a *app) routeBack() { a.step(layout.History.GoBack) }

func (a *app) routeForward() { a.step(layout.History.GoForward) }

// step moves through the route history, telling whether there was
// somewhere to go.
func (a *app) step(move func(layout.History, layout.Location) (layout.History, layout.Location, bool)) bool {
	moved := false
	a.quietly(func() {
		history, to, ok := move(a.route.history, a.location())
		if ok {
			a.route.history, moved = history, true
			a.goTo(to)
		}
	})
	return moved
}

// goTo makes the location current and focuses it, as revealSession does.
func (a *app) goTo(l layout.Location) {
	previous := a.state.ActiveProject
	region, err := a.state.RevealLocation(l)
	if err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(region)
	if a.state.ActiveProject != previous {
		a.checkCurrentDirectory()
	}
}
