package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) perspective(c *ui.Context, p *layout.Perspective, width float32) {
	ui.Column(c).Key(a.state.ActiveProject + "/" + string(a.state.CurrentProject().ActivePerspective)).Fill().Children(func() {
		top := func() { a.columns(c, slices.Clone(p.Columns), p.ColumnSplits, 0, width) }
		if p.BottomVisible {
			ui.SplitVerticalPercent(c, &p.TopPercent, top, func() { a.region(c, p.Bottom, false) }).Fill()
		} else {
			top()
		}
		ui.Row(c).Absolute().Top(4).Right(0).Width(38).Height(32).Background(a.theme.Surface).
			Children(func() { a.globalToggle(c) })
	})
}

func (a *app) columns(c *ui.Context, columns []*layout.Region, splits []float32, index int, width float32) {
	if len(columns) == 1 {
		a.region(c, columns[0], index == 0)
		return
	}
	constrainedSplit(c, &splits[index], width, minimumRegionWidth, columnsMinimum(len(columns)-1),
		func() { a.region(c, columns[0], index == 0) }, func() {
			a.columns(c, columns[1:], splits, index+1, (width-1)*(1-splits[index]/100))
		})
}

func (a *app) perspectiveSelect(c *ui.Context) {
	p := a.state.CurrentProject()
	selected := a.perspectiveName(p.ActivePerspective)
	options := []string{a.perspectiveName(layout.PerspectiveWork), a.perspectiveName(layout.PerspectiveReview)}
	ui.Select(c, &selected, options).Width(110).Shrink(0).Label(a.text("Perspective", "视角")).OnChange(func() {
		kind := layout.PerspectiveWork
		if selected == options[1] {
			kind = layout.PerspectiveReview
		}
		a.report(a.state.SelectPerspective(kind))
	})
}

// 200 DIPs keeps the perspective selector and region menu usable.
const minimumRegionWidth float32 = 200

func columnsMinimum(count int) float32 {
	return float32(count)*minimumRegionWidth + float32(count-1)
}

func constrainedSplit(c *ui.Context, percent *float32, width, firstMinimum, secondMinimum float32, first, second func()) ui.Element {
	constrain := func() {
		room := max(0, width-1)
		if room <= firstMinimum+secondMinimum {
			*percent = 100 * firstMinimum / (firstMinimum + secondMinimum)
			return
		}
		*percent = min(max(*percent, 100*firstMinimum/room), 100*(room-secondMinimum)/room)
	}
	constrain()
	return ui.SplitPercent(c, percent, first, second).Fill().OnChange(constrain)
}
