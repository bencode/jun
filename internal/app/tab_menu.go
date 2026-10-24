package app

import (
	"fmt"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) tabMenu(m *ui.Menu, region layout.RegionID, view layout.ViewID) {
	r := a.state.Region(region)
	if r == nil {
		return
	}
	if m.Item(a.text("Close Tab", "关闭标签")).Chosen() {
		a.closeView(region, view)
	}
	m.Submenu(a.text("Move Tab To", "移动标签到"), func(m *ui.Menu) { a.moveMenu(m, r, view) })
}

func (a *app) moveMenu(m *ui.Menu, source *layout.Region, view layout.ViewID) {
	target := func(r *layout.Region, label string) {
		if m.Item(label).Disabled(source == r).Chosen() {
			a.moveView(view, r.ID, 0)
		}
	}
	if p := a.state.CurrentPerspective(); p != nil {
		project := a.state.CurrentProject()
		prefix := project.Name + " / " + a.perspectiveName(project.ActivePerspective) + " / "
		for i, r := range p.Columns {
			target(r, prefix+fmt.Sprintf(a.text("Column %d", "第 %d 列"), i+1))
		}
		target(p.Bottom, prefix+a.text("Bottom Region", "底部区域"))
	}
	target(a.state.Global, "Global")
}
