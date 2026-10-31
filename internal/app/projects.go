package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) projectRailWidth() float32 {
	if a.state.RailExpanded {
		return 192
	}
	return 56
}

func (a *app) projectRail(c *ui.Context) {
	rail := ui.Column(c).Key("project-rail").Width(a.projectRailWidth()).Shrink(0).FillHeight().Padding(8).Gap(8)
	background := a.theme.Surface
	if rail.FileDragOver() {
		background = a.theme.Selection
	}
	rail.Background(background).Children(func() {
		ui.Button(c, logo).Height(36).FillWidth().Label(a.text("Toggle project names", "展开或收起项目名称")).
			Tooltip(a.text("Toggle project names", "展开或收起项目名称")).
			OnClick(func() { a.state.RailExpanded = !a.state.RailExpanded; a.layoutChanged() })
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				for _, p := range a.state.Projects {
					label := projectIcon(p.Name)
					if a.state.RailExpanded {
						label += "   " + p.Name
					}
					b := ui.Button(c.Key(p.Name), label).Height(36).FillWidth().Label(p.Name).Tooltip(p.Name + " " + p.CWD)
					if p.Name == a.state.ActiveProject {
						b.Background(a.theme.Selection).TextColor(a.theme.Text)
					}
					b.OnClick(func() { a.selectProject(p.Name) }).ContextMenu(func(m *ui.Menu) { a.projectMenu(m, p.Name) })
				}
				ui.Button(c, "+").Height(36).FillWidth().Label(a.text("Add Project", "添加项目")).
					Tooltip(a.text("Add Project", "添加项目")).OnClick(a.chooseProject)
			})
		})
		a.perspectiveButton(c)
		a.settingsButton(c)
	})
	if paths := rail.DroppedFiles(); paths != nil {
		a.openDroppedProject(paths)
		c.Invalidate()
	}
}

// openDroppedProject opens a directory dropped on the rail, as from Finder.
// Files are left for the views that will show them.
func (a *app) openDroppedProject(paths []string) {
	if len(paths) != 1 {
		a.report(errors.New(a.text("Drop one directory at a time.", "一次只能拖入一个目录。")))
		return
	}
	info, err := os.Stat(paths[0])
	if err != nil {
		a.report(err)
		return
	}
	if !info.IsDir() {
		a.report(fmt.Errorf(a.text("Only a directory opens as a project: %s", "只有目录可以作为项目打开：%s"), paths[0]))
		return
	}
	a.openProjectDirectory(paths[0])
}

// openProjectDirectory selects the project open at the directory, or
// opens one there, asking for a name only when the directory's name
// cannot be used as is.
func (a *app) openProjectDirectory(path string) {
	path = filepath.Clean(path)
	for _, p := range a.state.Projects {
		if p.CWD == path {
			a.selectProject(p.Name)
			return
		}
	}
	a.projectForm.path = path
	a.projectForm.name = filepath.Base(path)
	a.submitProject()
	a.projectForm.open = a.projectForm.error != ""
}

func (a *app) chooseProject() {
	window := a.window
	title := a.text("Open Project", "打开项目")
	// A blocking picker inside a GPU build pass prevents the sheet from painting.
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: window, Directory: true, Title: title,
		})
		window.Update(func() {
			if err != nil {
				a.report(err)
				return
			}
			if len(paths) == 0 {
				return
			}
			a.openProjectDirectory(paths[0])
		})
	}()
}

func (a *app) projectDialog(c *ui.Context) {
	ui.Modal(c, &a.projectForm.open, func() {
		ui.Column(c).Width(440).Gap(12).Children(func() {
			ui.Text(c, a.text("Open Project", "打开项目")).FontSize(18).Bold()
			ui.Text(c, a.projectForm.path).TextColor(a.theme.TextMuted)
			ui.TextInput(c, &a.projectForm.name).Label(a.text("Project name", "项目名称")).AutoFocus().FillWidth().OnSubmit(a.submitProject)
			if a.projectForm.error != "" {
				ui.Text(c, a.projectForm.error).TextColor(a.theme.Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).FillWidth().Children(func() {
				ui.Button(c, a.text("Cancel", "取消")).OnClick(func() { a.projectForm.open = false })
				ui.PrimaryButton(c, a.text("Open", "打开")).OnClick(a.submitProject)
			})
		})
	})
}

func (a *app) submitProject() {
	a.projectForm.error = ""
	name := strings.TrimSpace(a.projectForm.name)
	if name == "" {
		a.projectForm.error = a.text("Enter a project name.", "请输入项目名称。")
		return
	}
	for _, p := range a.state.Projects {
		if strings.EqualFold(p.Name, name) {
			a.projectForm.error = a.text("That project name already exists.", "项目名称已存在。")
			return
		}
	}
	cwd, err := filepath.Abs(a.projectForm.path)
	if err != nil {
		a.projectFailure(err)
		return
	}
	info, err := os.Stat(cwd)
	if err != nil {
		a.projectFailure(err)
		return
	}
	if !info.IsDir() {
		a.projectFailure(fmt.Errorf("not a directory: %s", cwd))
		return
	}
	if err := a.state.AddProject(name, cwd); err != nil {
		a.projectFailure(err)
		return
	}
	a.projectForm.open = false
	a.layoutChanged()
	a.focusCurrentPerspective()
	view := a.state.CurrentPerspective().Columns[0].Views[0]
	if err := a.startTerminal(view); err != nil {
		a.report(err)
	}
}

func (a *app) projectFailure(err error) {
	log.Printf("open project: %v", err)
	a.projectForm.error = a.text("The directory could not be opened: ", "无法打开目录：") + err.Error()
}
