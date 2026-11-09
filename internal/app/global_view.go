package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

// globalPane shows Global's views together, as GlobalArrange says. The room
// is its width.
func (a *app) globalPane(c *ui.Context, room float32) {
	views, arrange := a.state.GlobalViews(), a.state.GlobalArrange
	// A menu choice can change them during this build; keep focus pending for its next frame.
	current := func() bool {
		return a.state.GlobalVisible && a.state.GlobalArrange == arrange && slices.Equal(a.state.GlobalViews(), views)
	}
	a.globalTiles(c, views, room, func(v *layout.View) { a.globalView(c, v, v == views[0], current) })
}

// globalTiles splits the room between the first view and the rest, as
// columns splits a perspective's. A dragged column keeps its width as room
// changes; temporary constraints leave the saved width and shares alone.
func (a *app) globalTiles(c *ui.Context, views []*layout.View, room float32, tile func(*layout.View)) {
	if len(views) == 1 {
		tile(views[0])
		return
	}
	rest := views[1:]
	percent := layout.FirstShare(views)
	if a.state.GlobalArrange == layout.GlobalSide && views[0].Width > 0 && room > 1 {
		percent = 100 * views[0].Width / (room - 1)
	}
	commit := func() {
		layout.ResizeGlobalView(views, percent)
		a.layoutChanged()
	}
	first := func() { tile(views[0]) }
	if a.state.GlobalArrange == layout.GlobalStacked {
		ui.SplitVerticalPercent(c, &percent, first, func() { a.globalTiles(c, rest, room, tile) }).Fill().OnChange(commit)
		return
	}
	minimum := restMinimum(views, room)
	// Registered after constrainedSplit's own, so it sees the percentage
	// constrained. In a room too small for both, the divider cannot move.
	constrainedSplit(c, &percent, room, minimumRegionWidth, minimum, first, func() {
		a.globalTiles(c, rest, (room-1)*(1-percent/100), tile)
	}, func() {}).OnChange(func() {
		if room-1 > minimumRegionWidth+minimum {
			views[0].Width = (room - 1) * percent / 100
			commit()
		}
	})
}

// restMinimum is what the views after the first need: the next its minimum,
// and those after it their width, for a divider to move only its neighbors.
func restMinimum(views []*layout.View, room float32) float32 {
	if len(views) < 3 {
		return minimumRegionWidth
	}
	far := (room - 1) * layout.TotalShare(views[2:]) / layout.TotalShare(views)
	return max(columnsMinimum(len(views)-1), minimumRegionWidth+1+far)
}

// globalView is a view of Global under its title. The first shown has the
// region's menu, whose focus is Global's but no view's.
func (a *app) globalView(c *ui.Context, v *layout.View, first bool, current func() bool) {
	r := a.state.Global
	inView := false
	pane := ui.Column(c.Key(v.ID)).Fill().MinWidth(0).Clip().Children(func() {
		inView = a.globalHeader(c, v, first, current)
		if current() {
			body := ui.Column(c).Key("body").Grow(1).FillWidth().MinHeight(0)
			body.Children(func() { a.viewContent(c, v) })
			inView = inView || body.FocusWithin()
		}
	})
	if !current() || !pane.FocusWithin() || a.pendingFocus == r.ID {
		return
	}
	a.activeRegion = r.ID
	if inView && r.Active != v.ID {
		r.Active = v.ID
		a.layoutChanged()
	}
}

// globalHeader is a view's title row. It reports whether the title has the
// focus.
func (a *app) globalHeader(c *ui.Context, v *layout.View, first bool, current func() bool) bool {
	r := a.state.Global
	active := v.ID == r.Active
	focused := false
	ui.Box(c).Height(40).FillWidth().Shrink(0).Background(a.theme.Surface).Children(func() {
		ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(1).Background(a.theme.Border)
		ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).AlignItems(ui.Center).Gap(2).Children(func() {
			title := ui.ButtonBase(c).Grow(1).MinWidth(0).FillHeight().AlignItems(ui.Center).Padding(0, 10).Label(a.title(v))
			title.OnClick(func() { a.selectTab(r, v.ID) })
			if a.pendingFocus == r.ID && active && current() {
				title.Focus()
				a.pendingFocus = 0
			}
			title.Children(func() {
				text := ui.Text(c, a.title(v)).MinWidth(0).SingleLine().Ellipsis("…")
				if !active {
					text.TextColor(a.theme.TextMuted)
				}
			})
			focused = title.FocusWithin()
			if first {
				a.regionMenu(c, r)
			}
		})
		// Last, as a tab's mark is, so that it paints over the title.
		if active && r.ID == a.activeRegion {
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(2).Background(a.theme.Accent)
		}
	})
	return focused
}
