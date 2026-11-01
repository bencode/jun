package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) perspective(c *ui.Context, p *layout.Perspective, width float32) {
	ui.Column(c).Key(p.ID).Fill().Children(func() {
		main := func(width float32) {
			top := func() { a.columns(c, slices.Clone(p.Columns), p.ColumnSplits, 0, width) }
			if p.BottomVisible {
				ui.SplitVerticalPercent(c, &p.TopPercent, top, func() { a.region(c, p.Bottom) }).Fill().OnChange(a.layoutChanged)
			} else {
				top()
			}
		}
		if side := p.SideVisible; side {
			room, columns := width-1, columnsMinimum(len(p.Columns))
			shown := fitSide(p.SideWidth, room, columns)
			ui.Split(c, &shown, func() { a.region(c, p.Side) }, func() {
				// The side region's own menu can hide it during this build.
				if a.state.CurrentPerspective() == p && p.SideVisible == side {
					main(room - shown)
				}
			}).Fill().OnChange(func() {
				p.SideWidth = fitSide(shown, room, columns)
				a.layoutChanged()
			})
		} else {
			main(width)
		}
		ui.Row(c).Absolute().Top(4).Right(4).Width(32).Height(32).Background(a.theme.Surface).
			Children(func() { a.globalToggle(c) })
	})
}

func (a *app) columns(c *ui.Context, columns []*layout.Region, splits []float32, index int, width float32) {
	if len(columns) == 1 {
		a.region(c, columns[0])
		return
	}
	p := a.state.CurrentPerspective()
	count, bottom, side := len(p.Columns), p.BottomVisible, p.SideVisible
	constrainedSplit(c, &splits[index], width, minimumRegionWidth, columnsMinimum(len(columns)-1),
		func() { a.region(c, columns[0]) }, func() {
			if a.state.CurrentPerspective() != p || len(p.Columns) != count || p.BottomVisible != bottom || p.SideVisible != side {
				return
			}
			a.columns(c, columns[1:], splits, index+1, (width-1)*(1-splits[index]/100))
		}, a.layoutChanged)
}

// 200 DIPs keeps tab and region actions usable in narrow columns.
const minimumRegionWidth float32 = 200

func columnsMinimum(count int) float32 {
	return float32(count)*minimumRegionWidth + float32(count-1)
}

// fitSide is the width a side region of width takes in room, beside
// columns needing columnsMinimum: its own, within the minimums, or, with
// room short for both, a share in proportion to them.
func fitSide(width, room, columnsMinimum float32) float32 {
	if room <= minimumRegionWidth+columnsMinimum {
		return max(0, room) * minimumRegionWidth / (minimumRegionWidth + columnsMinimum)
	}
	return min(max(width, minimumRegionWidth), room-columnsMinimum)
}

func constrainedSplit(c *ui.Context, percent *float32, width, firstMinimum, secondMinimum float32, first, second, changed func()) ui.Element {
	constrain := func() {
		previous := *percent
		room := max(0, width-1)
		if room <= firstMinimum+secondMinimum {
			*percent = 100 * firstMinimum / (firstMinimum + secondMinimum)
		} else {
			*percent = min(max(*percent, 100*firstMinimum/room), 100*(room-secondMinimum)/room)
		}
		if *percent != previous {
			changed()
		}
	}
	constrain()
	return ui.SplitPercent(c, percent, first, second).Fill().OnChange(func() { constrain(); changed() })
}
