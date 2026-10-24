package app

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	c.SetTheme(a.theme)
	ui.Row(c).Fill().Gap(0).AlignItems(ui.Stretch).Children(func() {
		a.projectRail(c)
		width, _ := c.Size()
		width -= a.projectRailWidth()
		if a.state.GlobalVisible {
			centerMinimum := minimumRegionWidth
			if p := a.state.CurrentPerspective(); p != nil {
				centerMinimum = columnsMinimum(len(p.Columns))
			}
			center := func() { a.center(c, (width-1)*a.state.CenterPercent/100) }
			constrainedSplit(c, &a.state.CenterPercent, width, centerMinimum, minimumRegionWidth, center, func() {
				a.region(c, a.state.Global)
			}, a.layoutChanged).Grow(1).MinWidth(0).FillHeight()
		} else {
			ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() { a.center(c, width) })
		}
	})
	a.shortcuts(c)
	a.projectDialog(c)
	a.perspectiveDialog(c)
	ui.Modal(c, &a.errorOpen, func() {
		ui.Text(c, a.text("Unable to complete action", "无法完成操作")).Bold()
		ui.Text(c, a.errorMessage)
		ui.Button(c, a.text("OK", "确定")).OnClick(func() { a.errorOpen = false })
	})
	a.persistenceDialog(c)
	a.autosave(c)
}

func (a *app) center(c *ui.Context, width float32) {
	if p := a.state.CurrentPerspective(); p != nil {
		a.perspective(c, p, width)
		return
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(40).FillWidth().Justify(ui.End).Children(func() { a.globalToggle(c) })
		ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Gap(12).Children(func() {
			ui.Text(c, a.text("Your agents. Your perspective.", "你的 Agents，你的工作视角。")).FontSize(23).Bold()
			ui.Text(c, a.text("Open a directory to start a project.", "打开目录，开始一个项目。")).TextColor(a.theme.TextMuted)
			ui.PrimaryButton(c, a.text("Open Project…", "打开项目…")).OnClick(a.chooseProject)
		})
	})
}

func (a *app) globalToggle(c *ui.Context) {
	label := a.text("Show Global", "展开 Global")
	if a.state.GlobalVisible {
		label = a.text("Hide Global", "收起 Global")
	}
	ui.Button(c, globalIcon).Label(label).Tooltip(label).Width(36).Height(32).
		OnClick(a.toggleGlobal)
}
