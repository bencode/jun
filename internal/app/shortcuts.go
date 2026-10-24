package app

import "github.com/egoist/mygo/ui"

func (a *app) shortcuts(c *ui.Context) {
	c.OnShortcut(ui.Super, ui.KeyT, func() {
		if r := a.targetRegion(); r != nil {
			a.newTerminal(r.ID)
		}
	})
	c.OnShortcut(ui.Super, ui.KeyW, func() {
		if r := a.targetRegion(); r != nil && r.Active != 0 {
			a.closeView(r.ID, r.Active)
		}
	})
	c.OnShortcut(ui.Ctrl, ui.KeyTab, func() { a.cycleTab(1) })
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyTab, func() { a.cycleTab(-1) })
}
