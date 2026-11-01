package app

import (
	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

// openViewMenu offers the views a column or bottom region can open; those
// of Global and the side regions are fixed.
func (a *app) openViewMenu(m *ui.Menu, target layout.RegionID) {
	noProject := a.state.CurrentProject() == nil
	if m.Item(a.text("New Terminal", "新建终端")).Shortcut(ui.Super, ui.KeyT).Disabled(noProject).Chosen() {
		a.newTerminal(target)
	}
}
