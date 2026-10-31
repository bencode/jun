package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

func (a *app) openViewMenu(m *ui.Menu, target layout.RegionID) {
	noProject := a.state.CurrentProject() == nil
	if m.Item(a.text("New Terminal", "新建终端")).Shortcut(ui.Super, ui.KeyT).Disabled(noProject).Chosen() {
		a.newTerminal(target)
	}
	if m.Item("Agents").Chosen() {
		a.openPreview(target, layout.ViewAgents)
	}
	if m.Item(a.text("Files (Preview)", "文件树（预览）")).Disabled(noProject).Chosen() {
		a.openPreview(target, layout.ViewTree)
	}
}

func (a *app) openPreview(target layout.RegionID, kind layout.ViewKind) {
	if kind != layout.ViewAgents && kind != layout.ViewTree {
		a.report(fmt.Errorf("unsupported preview: %s", kind))
		return
	}
	r := a.state.Region(target)
	if r == nil {
		a.report(fmt.Errorf("region not found: %d", target))
		return
	}
	projectName := ""
	if kind == layout.ViewTree {
		p := a.state.CurrentProject()
		if p == nil {
			return
		}
		projectName = p.Name
	}
	index := slices.IndexFunc(r.Views, func(v *layout.View) bool { return v.Kind == kind && v.ProjectName == projectName })
	if index >= 0 {
		view := r.Views[index]
		if r.Active != view.ID {
			if err := a.state.SelectView(target, view.ID); err != nil {
				a.report(err)
				return
			}
			a.layoutChanged()
		}
		a.focusRegion(target)
		return
	}
	if _, err := a.state.OpenView(target, kind, projectName); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(target)
}
