package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

func (a *app) projectMenu(m *ui.Menu, name string) {
	if m.Item(a.text("Rename Project…", "重命名项目…")).Chosen() {
		f := &a.projectRename
		f.open, f.project, f.name, f.error = true, name, name, ""
	}
	if m.Item(a.text("Remove Project…", "移除项目…")).Chosen() {
		a.confirmRemoveProject(name)
	}
}

func (a *app) projectRenameDialog(c *ui.Context) {
	f := &a.projectRename
	cancel := func() {
		f.open = false
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	}
	modal := ui.Modal(c, &f.open, func() {
		ui.Column(c).Width(400).Gap(12).Children(func() {
			ui.Text(c, a.text("Rename Project", "重命名项目")).FontSize(theme.Title).Bold()
			ui.Text(c, f.project).TextColor(a.theme.TextMuted)
			ui.TextInput(c, &f.name).Label(a.text("Project name", "项目名称")).AutoFocus().FillWidth().OnSubmit(a.submitProjectRename)
			if f.error != "" {
				ui.Text(c, f.error).TextColor(a.theme.Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).FillWidth().Children(func() {
				ui.Button(c, a.text("Cancel", "取消")).OnClick(cancel)
				ui.PrimaryButton(c, a.text("Rename", "重命名")).OnClick(a.submitProjectRename)
			})
		})
	})
	if modal.Dismissed() {
		cancel()
	}
}

func (a *app) submitProjectRename() {
	f := &a.projectRename
	name := strings.TrimSpace(f.name)
	if name == "" {
		f.error = a.text("Enter a project name.", "请输入项目名称。")
		return
	}
	if slices.ContainsFunc(a.state.Projects, func(p *layout.Project) bool { return p.Name != f.project && strings.EqualFold(p.Name, name) }) {
		f.error = a.text("That project name already exists.", "项目名称已存在。")
		return
	}
	if err := a.state.RenameProject(f.project, name); err != nil {
		f.error = err.Error()
		return
	}
	a.terminals.Rename(f.project, name)
	if name != f.project {
		a.layoutChanged()
	}
	f.open, f.error = false, ""
	if r := a.targetRegion(); r != nil {
		a.focusRegion(r.ID)
	}
}

func (a *app) confirmRemoveProject(name string) {
	count := len(slices.DeleteFunc(a.terminals.ForProject(name), (*terminal.Session).Exited))
	detail := a.text("The directory stays on disk.", "磁盘上的目录不受影响。")
	if count > 0 {
		detail = fmt.Sprintf(a.text("Its terminals (%d) will be closed, together with any agents running in them. The directory stays on disk.",
			"将关闭它的 %d 个终端，其中运行的 agent 也会一起关掉。磁盘上的目录不受影响。"), count)
	}
	a.confirm(fmt.Sprintf(a.text("Remove project “%s”?", "移除项目「%s」？"), name), detail,
		a.text("Remove", "移除"), func() { a.removeProject(name) })
}

// removeProject closes all of the project's sessions, hidden ones too:
// with the project gone, none could be reopened.
func (a *app) removeProject(name string) {
	previous := a.state.ActiveProject
	views, err := a.state.RemoveProject(name)
	if err != nil {
		a.report(err)
		return
	}
	a.dropViews(views)
	var sessions []terminal.ID
	for _, s := range a.terminals.ForProject(name) {
		sessions = append(sessions, s.ID)
	}
	a.report(a.terminals.Close(sessions...))
	a.layoutChanged()
	a.focusCurrentPerspective()
	if a.state.ActiveProject != previous {
		a.checkCurrentDirectory()
	}
}
