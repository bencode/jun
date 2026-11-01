package app

import (
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	a.focusedTerminal = 0
	c.SetTheme(a.theme)
	ui.Row(c).Fill().Gap(0).AlignItems(ui.Stretch).Children(func() {
		a.projectRail(c)
		width, _ := c.Size()
		width -= a.projectRailWidth()
		if a.state.GlobalVisible {
			centerMinimum := minimumRegionWidth
			if p := a.state.CurrentPerspective(); p != nil {
				centerMinimum = columnsMinimum(len(p.Columns))
				if p.SideVisible {
					centerMinimum += minimumRegionWidth + 1
				}
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
	a.paletteDialog(c)
	a.projectDialog(c)
	a.perspectiveDialog(c)
	a.projectRenameDialog(c)
	a.confirmDialog(c)
	ui.Modal(c, &a.errorOpen, func() {
		ui.Text(c, a.text("Unable to complete action", "无法完成操作")).Bold()
		ui.Text(c, a.errorMessage)
		ui.Button(c, a.text("OK", "确定")).OnClick(func() { a.errorOpen = false })
	})
	a.persistenceDialog(c)
	if a.reapExitedTerminals(a.focusedTerminal) {
		c.Invalidate()
	}
	a.autosave(c)
}

func (a *app) center(c *ui.Context, width float32) {
	if p := a.state.CurrentPerspective(); p != nil {
		a.perspective(c, p, width)
		return
	}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(40).FillWidth().Justify(ui.End).AlignItems(ui.Center).Padding(0, 7).Children(func() { a.globalToggle(c) })
		ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Gap(12).Children(func() {
			ui.Icon(c, logoIcon).Height(48).TextColor(a.theme.Accent)
			ui.Text(c, a.text("Your agents. Your perspective.", "你的 Agents，你的工作视角。")).FontSize(theme.Heading).Bold()
			ui.Text(c, a.text("Open a directory to start a project.", "打开目录，开始一个项目。")).TextColor(a.theme.TextMuted)
			ui.PrimaryButton(c, a.text("Open Project…", "打开项目…")).OnClick(func() { a.openPalette("~/") })
		})
	})
}
