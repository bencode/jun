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
			ui.SplitVerticalPercent(c, &p.TopPercent, top, func() { a.region(c, p.Bottom, false) }).Fill().OnChange(a.layoutChanged)
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
	p := a.state.CurrentPerspective()
	count, bottom := len(p.Columns), p.BottomVisible
	constrainedSplit(c, &splits[index], width, minimumRegionWidth, columnsMinimum(len(columns)-1),
		func() { a.region(c, columns[0], index == 0) }, func() {
			if a.state.CurrentPerspective() != p || len(p.Columns) != count || p.BottomVisible != bottom {
				return
			}
			a.columns(c, columns[1:], splits, index+1, (width-1)*(1-splits[index]/100))
		}, a.layoutChanged)
}

func (a *app) perspectiveSelect(c *ui.Context) ui.Element {
	p := a.state.CurrentProject()
	return ui.MenuButton(c, a.perspectiveName(p.ActivePerspective), a.perspectiveMenu).
		Width(110).Height(32).Shrink(0).Justify(ui.SpaceBetween).
		Label(a.text("Perspective", "视角")).Tooltip(a.text("Perspective and layout", "视角与布局"))
}

func (a *app) perspectiveMenu(m *ui.Menu) {
	project := a.state.CurrentProject()
	if project == nil {
		return
	}
	for _, kind := range []layout.PerspectiveKind{layout.PerspectiveWork, layout.PerspectiveReview} {
		if m.Item(a.perspectiveName(kind)).Checked(project.ActivePerspective == kind).Chosen() {
			a.selectPerspective(kind)
		}
	}
	p := a.state.CurrentPerspective()
	label := a.text("Show Bottom Region", "展开底部区域")
	if p.BottomVisible {
		label = a.text("Hide Bottom Region", "收起底部区域")
	}
	m.Separator()
	if m.Item(label).Chosen() {
		a.setBottomVisible(p.Bottom.ID, !p.BottomVisible)
	}
}

// 200 DIPs keeps the perspective selector and region menu usable.
const minimumRegionWidth float32 = 200

func columnsMinimum(count int) float32 {
	return float32(count)*minimumRegionWidth + float32(count-1)
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
