package app

import "github.com/egoist/mygo/ui"

// viewBar ends the last column's tab bar, top right of the perspective: the
// perspective's menu, then switches for the regions around the columns. It
// belongs to no region: using it leaves the active region as it was.
func (a *app) viewBar(c *ui.Context) ui.Element {
	p := a.state.CurrentPerspective()
	// In a narrow column the perspective's name gives way, down to its icon.
	return ui.Row(c.Key("view-bar")).Shrink(1).MinWidth(0).Height(32).Padding(0, 3).Gap(2).AlignItems(ui.Center).
		Background(a.theme.Surface).Children(func() {
		ui.ButtonBase(c.Key("perspective-switcher")).Height(26).Shrink(1).MinWidth(44).MaxWidth(170).Padding(0, 8).Gap(6).
			Label(a.text("Perspective", "视角")).Tooltip(a.state.ActiveProject + " / " + p.Name).Children(func() {
			ui.Icon(c, perspectiveIcon).Size(14, 14).Shrink(0)
			ui.Text(c, p.Name).MinWidth(0).SingleLine().Ellipsis("…")
			ui.Icon(c, downIcon).Size(11, 11).Shrink(0)
		}).Menu(a.perspectiveMenu)
		ui.Box(c).Width(1).Height(16).Shrink(0).Margin(0, 3).Background(a.theme.Border)
		side := a.text("Show Side Bar", "展开侧边栏")
		if p.SideVisible {
			side = a.text("Hide Side Bar", "收起侧边栏")
		}
		a.layoutToggle(c, sideIcon, side, side+" (⌘B)", p.SideVisible, a.toggleSide)
		bottom := a.text("Show Bottom Region", "展开底部区域")
		if p.BottomVisible {
			bottom = a.text("Hide Bottom Region", "收起底部区域")
		}
		a.layoutToggle(c, bottomIcon, bottom, bottom, p.BottomVisible, func() { a.setBottomVisible(p.Bottom.ID, !p.BottomVisible) })
		a.globalToggle(c)
	})
}

func (a *app) globalToggle(c *ui.Context) {
	label := a.text("Show Global", "展开 Global")
	if a.state.GlobalVisible {
		label = a.text("Hide Global", "收起 Global")
	}
	a.layoutToggle(c, globalIcon, label, label, a.state.GlobalVisible, a.toggleGlobal)
}

// layoutToggle switches a region, its icon drawing the region and lit
// while it shows.
func (a *app) layoutToggle(c *ui.Context, icon *ui.SVG, label, tip string, on bool, toggle func()) {
	button := ui.ButtonBase(c.Key(icon)).Width(28).Height(26).Shrink(0).Justify(ui.Center).Label(label).Tooltip(tip)
	if on {
		button.Background(a.theme.Selection).TextColor(a.theme.Accent)
	}
	button.Children(func() { ui.Icon(c, icon).Size(16, 16) }).OnClick(func() {
		toggle()
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	})
}
