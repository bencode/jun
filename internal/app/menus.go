package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) applicationMenu() *mygo.Menu {
	action := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(_ *mygo.MenuItem, _ *mygo.Window) {
			if a.window != nil {
				a.window.Update(fn)
			} else {
				fn()
			}
		}
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Jun", Submenu: []*mygo.MenuItem{{Role: mygo.RoleAbout}, mygo.Separator(), {Role: mygo.RoleQuit}}},
		{Label: a.text("File", "文件"), Submenu: []*mygo.MenuItem{
			{Label: a.text("Command Palette…", "命令面板…"), Accelerator: "CmdOrCtrl+K", Click: action(func() { a.openPalette("") })},
			{Label: a.text("Review Changes…", "查看改动…"), Accelerator: "CmdOrCtrl+Shift+D", Click: action(a.reviewChanges)},
			{Label: a.text("Open Project…", "打开项目…"), Accelerator: "CmdOrCtrl+O", Click: action(func() { a.openPalette("~/") })},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: a.text("View", "视图"), Submenu: []*mygo.MenuItem{
			{Label: a.text("Toggle Global", "展开或收起 Global"), Click: action(a.toggleGlobal)},
			{Label: a.text("Toggle Side Bar", "展开或收起侧边栏"), Click: action(a.toggleSide)},
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Label: a.text("Language", "语言"), Submenu: []*mygo.MenuItem{
			{Label: "English", Type: mygo.MenuItemCheckbox, Checked: !a.state.Chinese, Click: action(func() { a.setLanguage(false) })},
			{Label: "简体中文", Type: mygo.MenuItemCheckbox, Checked: a.state.Chinese, Click: action(func() { a.setLanguage(true) })},
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

func (a *app) regionMenu(c *ui.Context, r *layout.Region) {
	button := iconButton(c, moreIcon, fmt.Sprintf("%s %d", a.text("Region menu", "区域菜单"), r.ID), 32, 16).
		Tooltip(a.text("Region actions", "区域操作"))
	if a.pendingFocus == r.ID && r.Active == 0 {
		button.Focus()
		a.pendingFocus = 0
	}
	button.Menu(func(m *ui.Menu) {
		if r == a.state.Global {
			if m.Item(a.text("Hide Global", "收起 Global")).Chosen() {
				a.toggleGlobal()
			}
			return
		}
		if p := a.state.CurrentPerspective(); p != nil && r == p.Side {
			if m.Item(a.text("Hide Side Bar", "收起侧边栏")).Shortcut(ui.Super, ui.KeyB).Chosen() {
				a.toggleSide()
			}
			return
		}
		m.Submenu(a.text("Open View", "打开视图"), func(m *ui.Menu) { a.openViewMenu(m, r.ID) })
		m.Submenu(a.text("Reopen Terminal", "重新打开终端"), func(m *ui.Menu) {
			projectName := ""
			if p := a.state.CurrentProject(); p != nil {
				projectName = p.Name
			}
			sessions := a.hiddenTerminals(projectName)
			if len(sessions) == 0 {
				m.Item(a.text("No closed terminals", "没有已关闭的终端")).Disabled(true)
			}
			for _, session := range sessions {
				label := fmt.Sprintf("%s · #%d", a.sessionTitle(session), session.ID)
				if m.Item(label).Chosen() {
					a.reopenTerminal(r.ID, session.ID)
				}
			}
		})
		if p := a.state.CurrentPerspective(); p != nil {
			if slices.Contains(p.Columns, r) {
				m.Separator()
				a.columnMenu(m, r, p)
			}
			if r == p.Bottom && m.Item(a.text("Hide Bottom Region", "收起底部区域")).Chosen() {
				a.setBottomVisible(r.ID, false)
			}
		}
	})
}

func (a *app) columnMenu(m *ui.Menu, r *layout.Region, p *layout.Perspective) {
	if m.Item(a.text("Add Column to Right", "在右侧添加列")).Disabled(len(p.Columns) >= 3).Chosen() {
		a.addColumn(r.ID)
	}
	if m.Item(a.text("Remove This Column", "移除此列")).Disabled(len(p.Columns) == 1).Chosen() {
		a.removeColumn(r.ID)
	}
}
