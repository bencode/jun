package app

import "github.com/egoist/mygo/ui"

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
	if m.Item(a.text("Delete Perspective…", "删除视角…")).Disabled(len(project.Perspectives) == 1).Chosen() {
		a.confirmDeletePerspective(a.state.CurrentPerspective())
	}
}
