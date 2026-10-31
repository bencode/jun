package app

import "github.com/egoist/mygo/ui"

func (a *app) perspectiveButton(c *ui.Context) {
	p := a.state.CurrentPerspective()
	name := a.text("Perspective", "视角")
	tip := a.text("Open a project to use perspectives", "打开项目后使用视角")
	if p != nil {
		name = p.Name
		tip = a.state.ActiveProject + " / " + p.Name
	}
	ui.ButtonBase(c.Key("perspective-switcher")).Height(36).FillWidth().Shrink(0).Padding(0, 10).Gap(8).
		Label(a.text("Perspective", "视角")).Tooltip(tip).Disabled(p == nil).Children(func() {
		ui.Icon(c, perspectiveIcon).Size(18, 18).Shrink(0)
		if a.state.RailExpanded {
			ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().Ellipsis("…")
			ui.Icon(c, downIcon).Size(12, 12).Shrink(0)
		}
	}).Menu(a.perspectiveMenu)
}

func (a *app) perspectiveMenu(m *ui.Menu) {
	project := a.state.CurrentProject()
	if project == nil {
		return
	}
	m.Item(project.Name + " · " + a.text("Perspectives", "视角")).Disabled(true)
	m.Separator()
	for _, p := range project.Perspectives {
		if m.Item(p.Name).Checked(project.ActivePerspective == p.ID).Chosen() {
			a.selectPerspective(p.ID)
		}
	}
	m.Separator()
	if m.Item(a.text("New Perspective…", "新建视角…")).Chosen() {
		a.openPerspectiveForm(0)
	}
	if m.Item(a.text("Rename Perspective…", "重命名视角…")).Chosen() {
		a.openPerspectiveForm(project.ActivePerspective)
	}
	p := a.state.CurrentPerspective()
	if m.Item(a.text("Delete Perspective…", "删除视角…")).Disabled(len(project.Perspectives) == 1).Chosen() {
		a.confirmDeletePerspective(p)
	}
	label := a.text("Show Bottom Region", "展开底部区域")
	if p.BottomVisible {
		label = a.text("Hide Bottom Region", "收起底部区域")
	}
	m.Separator()
	if m.Item(label).Chosen() {
		a.setBottomVisible(p.Bottom.ID, !p.BottomVisible)
	}
}
