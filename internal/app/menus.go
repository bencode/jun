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
	language := func(chinese bool) {
		a.chinese = chinese
		mygo.App.SetMenu(a.applicationMenu())
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "Jun", Submenu: []*mygo.MenuItem{{Role: mygo.RoleAbout}, mygo.Separator(), {Role: mygo.RoleQuit}}},
		{Label: a.text("File", "文件"), Submenu: []*mygo.MenuItem{
			{Label: a.text("Open Project…", "打开项目…"), Accelerator: "CmdOrCtrl+O", Click: action(a.chooseProject)},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: a.text("View", "视图"), Submenu: []*mygo.MenuItem{
			{Label: a.text("Toggle Global", "展开或收起 Global"), Click: action(func() { a.state.GlobalVisible = !a.state.GlobalVisible })},
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Label: a.text("Language", "语言"), Submenu: []*mygo.MenuItem{
			{Label: "English", Type: mygo.MenuItemCheckbox, Checked: !a.chinese, Click: action(func() { language(false) })},
			{Label: "简体中文", Type: mygo.MenuItemCheckbox, Checked: a.chinese, Click: action(func() { language(true) })},
		}},
		{Role: mygo.RoleWindowMenu},
	})
}

func (a *app) regionMenu(c *ui.Context, r *layout.Region) {
	button := ui.Button(c, moreIcon).Width(30).Height(32).Shrink(0).
		Label(fmt.Sprintf("%s %d", a.text("Region menu", "区域菜单"), r.ID)).Tooltip(a.text("Region actions", "区域操作"))
	if a.pendingFocus == r.ID && r.Active == 0 {
		button.Focus()
		a.pendingFocus = 0
	}
	button.Menu(func(m *ui.Menu) {
		active := r.Active
		if m.Item(a.text("New Terminal", "新建终端")).Disabled(a.state.CurrentProject() == nil).Chosen() {
			a.newTerminal(r.ID)
		}
		if m.Item(a.text("Close View", "关闭视图")).Disabled(active == 0).Chosen() {
			a.closeView(r.ID, active)
		}
		m.Submenu(a.text("Move View To", "移动视图到"), func(m *ui.Menu) { a.moveMenu(m, r, active) })
		m.Separator()
		if p := a.state.CurrentPerspective(); p != nil {
			a.columnMenu(m, r, p)
		}
		if r == a.state.Global {
			hasAgents := slices.ContainsFunc(r.Views, func(v *layout.View) bool { return v.Kind == layout.ViewAgents })
			if m.Item(a.text("Open Agents", "打开 Agents")).Disabled(hasAgents).Chosen() {
				_, err := a.state.OpenView(r.ID, layout.ViewAgents, "")
				a.report(err)
				a.pendingFocus = r.ID
			}
			if m.Item(a.text("Hide Global", "收起 Global")).Chosen() {
				a.state.GlobalVisible = false
			}
		}
	})
}

func (a *app) columnMenu(m *ui.Menu, r *layout.Region, p *layout.Perspective) {
	if m.Item(a.text("Add Column", "添加列")).Disabled(len(p.Columns) >= 3).Chosen() {
		id, err := a.state.AddColumn()
		a.report(err)
		if err == nil {
			a.pendingFocus = id
		}
	}
	column := slices.Contains(p.Columns, r)
	if column && m.Item(a.text("Remove Column", "移除此列")).Disabled(len(p.Columns) == 1).Chosen() {
		index := slices.Index(p.Columns, r)
		err := a.state.RemoveColumn(r.ID)
		a.report(err)
		if err == nil {
			a.pendingFocus = p.Columns[max(0, index-1)].ID
		}
	}
	label := a.text("Show Bottom Region", "展开底部区域")
	if p.BottomVisible {
		label = a.text("Hide Bottom Region", "收起底部区域")
	}
	if m.Item(label).Chosen() {
		p.BottomVisible = !p.BottomVisible
	}
}

func (a *app) moveMenu(m *ui.Menu, source *layout.Region, active layout.ViewID) {
	target := func(r *layout.Region, label string) {
		if m.Item(label).Disabled(active == 0 || source == r).Chosen() {
			a.moveView(active, r.ID, 0)
		}
	}
	if p := a.state.CurrentPerspective(); p != nil {
		for i, r := range p.Columns {
			target(r, fmt.Sprintf(a.text("Column %d", "第 %d 列"), i+1))
		}
		target(p.Bottom, a.text("Bottom Region", "底部区域"))
	}
	target(a.state.Global, "Global")
}
