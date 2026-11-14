package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
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
	rail := ui.Column(c).Key("project-rail").Width(a.projectRailWidth()).Shrink(0).FillHeight()
	background := a.theme.Surface
	if rail.FileDragOver() {
		background = a.theme.Selection
	}
	rail.Background(background).Children(func() {
		a.railHeader(c)
		ui.Column(c).Grow(1).FillWidth().MinHeight(0).Padding(8).Gap(8).Children(func() {
			ui.Scroll(c).Grow(1).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Gap(6).Children(func() {
					// Without a home, paths show in full.
					home, _ := os.UserHomeDir()
					waiting := a.waitingByProject()
					panels := a.panelsByProject()
					shown := a.state.ShownProjects()
					for i, p := range shown {
						tip := p.Name + " " + p.CWD
						if keys, ok := projectKeys(i); ok {
							tip += "\n" + keys.String()
						}
						if !panels[p.Name] {
							tip += "\n" + a.text("No open panels", "没有打开的面板")
						}
						a.projectButton(c, p, panels[p.Name], waiting[p.Name], home, tip)
					}
					if shelved := a.state.ShelvedProjects(); len(shelved) > 0 {
						a.shelfButton(c, shelved, waiting)
					}
					add := a.text("Add Project", "添加项目")
					a.railAction(ui.ButtonBase(c.Key("add-project")).Height(36).FillWidth().Shrink(0).Justify(ui.Center).
						Label(add).Tooltip(add + " (" + openProjectKeys.String() + ")")).Children(func() {
						ui.Icon(c, addIcon).Size(18, 18).Shrink(0)
					}).OnClick(func() { a.openPalette("~/") })
				})
			})
			ui.Box(c).Height(1).FillWidth().Shrink(0).Background(a.theme.Border)
			a.settingsButton(c)
		})
	})
	if paths := rail.DroppedFiles(); paths != nil {
		a.openDroppedProject(paths)
		c.Invalidate()
	}
}

// railHeader heads the rail as the tab bars head the regions, its line
// continuing theirs: "Projects" and their number when the rail is
// expanded, and the button expanding or collapsing it.
func (a *app) railHeader(c *ui.Context) {
	expanded := a.state.RailExpanded
	label := a.text("Show Project Names", "展开项目名称")
	if expanded {
		label = a.text("Hide Project Names", "收起项目名称")
	}
	ui.Box(c).Height(40).FillWidth().Shrink(0).Children(func() {
		ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(1).Background(a.theme.Border)
		row := ui.Row(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).AlignItems(ui.Center).Gap(6)
		if expanded {
			row.Padding(0, 8, 0, 14)
		} else {
			row.Justify(ui.Center)
		}
		row.Children(func() {
			if expanded {
				ui.Text(c, a.text("Projects", "项目")).FontSize(theme.Caption).Bold().TextColor(a.theme.TextMuted).SingleLine()
				ui.Text(c, fmt.Sprint(len(a.state.Projects))).FontSize(theme.Caption).TextColor(a.theme.TextMuted).Grow(1)
			}
			a.railAction(ui.ButtonBase(c.Key("rail-toggle")).Width(28).Height(26).Shrink(0).Justify(ui.Center).
				Label(label).Tooltip(label)).Children(func() {
				ui.Icon(c, railIcon).Size(16, 16)
			}).OnClick(a.toggleRail)
		})
	})
}

func (a *app) toggleRail() {
	a.state.RailExpanded = !a.state.RailExpanded
	a.layoutChanged()
}

// projectButton shows a project as a colored tile of its abbreviation,
// with its name and path beside it when the rail is expanded, and marks
// the agents waiting in it; tip is its tooltip.
func (a *app) projectButton(c *ui.Context, p *layout.Project, hasPanels bool, waiting int, home, tip string) {
	expanded := a.state.RailExpanded
	height := a.railTileHeight()
	waitingLabel := a.waitingLabel(waiting)
	active := p.Name == a.state.ActiveProject
	b := ui.ButtonBase(c.Key(p.Name)).Height(height).FillWidth().Shrink(0).Padding(0, 4).Gap(8).Justify(ui.Start).
		Label(p.Name).Tooltip(tip).Children(func() {
		if active {
			ui.Box(c).Absolute().Left(0).Top((height - 16) / 2).Width(2).Height(16).Radius(1).Background(a.theme.Accent)
		}
		ui.Box(c).Width(32).Height(32).Shrink(0).Radius(8).Background(projectColor(p.Name)).Center().Children(func() {
			ui.Text(c, projectAbbrev(p.Name)).Bold().TextColor(ui.RGB(255, 255, 255))
			if waiting > 0 && !expanded {
				a.waitingDot(c, waitingLabel)
			}
		})
		if !expanded {
			return
		}
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, p.Name).SingleLine().Ellipsis("…")
			ui.Text(c, shortPath(p.CWD, home)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…")
		})
		if waiting > 0 {
			a.waitingCount(c, waiting, waitingLabel)
		}
	})
	if active {
		b.Background(a.theme.SurfaceHover)
	}
	if !hasPanels && !active && waiting == 0 && !b.Hovered() {
		b.Opacity(0.55)
	}
	b.OnClick(func() { a.selectProject(p.Name) }).ContextMenu(func(m *ui.Menu) { a.projectMenu(m, p.Name) })
}

// railTileHeight is the height of a rail tile, taller beside a name and
// path when the rail is expanded.
func (a *app) railTileHeight() float32 {
	if a.state.RailExpanded {
		return 44
	}
	return 40
}

func (a *app) waitingLabel(n int) string {
	return fmt.Sprintf(a.text("%d waiting", "%d 个等待中"), n)
}

// waitingDot marks a tile's corner when agents wait; waitingCount shows
// their number beside it in the expanded rail.
func (a *app) waitingDot(c *ui.Context, label string) {
	ui.Box(c).Absolute().Top(-3).Right(-3).Width(10).Height(10).Radius(5).
		Background(a.theme.Warning).Border(2, a.theme.Surface).Label(label)
}

func (a *app) waitingCount(c *ui.Context, n int, label string) {
	ui.Text(c, fmt.Sprintf("●%d", n)).FontSize(theme.Caption).Shrink(0).TextColor(a.theme.Warning).Label(label)
}

// panelsByProject excludes the fixed side views and follows a panel's
// source project even when it has moved into another project's layout.
func (a *app) panelsByProject() map[string]bool {
	panels := make(map[string]bool)
	for _, p := range a.state.Projects {
		for _, perspective := range p.Perspectives {
			for _, r := range perspective.Columns {
				for _, v := range r.Views {
					panels[v.ProjectName] = true
				}
			}
			for _, v := range perspective.Bottom.Views {
				panels[v.ProjectName] = true
			}
		}
	}
	return panels
}

// railAction quiets a secondary rail button until it is hovered.
func (a *app) railAction(b ui.Element) ui.Element {
	if b.Hovered() {
		return b.TextColor(a.theme.Text)
	}
	return b.TextColor(a.theme.TextMuted)
}

// shortPath shows home as "~" and keeps the last two parts of a long path.
func shortPath(cwd, home string) string {
	root, rest := "", strings.TrimPrefix(cwd, "/")
	if home != "" && (cwd == home || strings.HasPrefix(cwd, home+"/")) {
		root, rest = "~", strings.TrimPrefix(strings.TrimPrefix(cwd, home), "/")
	}
	var parts []string
	if rest != "" {
		parts = strings.Split(rest, "/")
	}
	if len(parts) > 3 {
		parts = append([]string{"…"}, parts[len(parts)-2:]...)
	}
	if root == "~" && len(parts) == 0 {
		return root
	}
	return root + "/" + strings.Join(parts, "/")
}

// openDroppedProject opens a directory dropped on the rail, as from Finder.
// Files are left for the views that will show them.
func (a *app) openDroppedProject(paths []string) {
	if len(paths) != 1 {
		a.report(errors.New(a.text("Drop one directory at a time.", "一次只能拖入一个目录。")))
		return
	}
	a.openProjectPath(paths[0])
}

// openProjectPath opens path as a project, or tells why a path that is not
// a directory cannot be one.
func (a *app) openProjectPath(path string) {
	info, err := os.Stat(path)
	if err != nil {
		a.report(err)
		return
	}
	if !info.IsDir() {
		a.report(fmt.Errorf(a.text("Only a directory opens as a project: %s", "只有目录可以作为项目打开：%s"), path))
		return
	}
	a.openProjectDirectory(path)
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

// openArgs opens what jun was started with, relative to wd: a directory
// as a project, as "jun ." in a terminal, and a file in a tab of the
// current project, read-only, as "jun tool open file:" does; flags, as
// macOS may pass, are neither.
func (a *app) openArgs(args []string, wd string) {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if !filepath.IsAbs(arg) {
			arg = filepath.Join(wd, arg)
		}
		if info, err := os.Stat(arg); err == nil && info.Mode().IsRegular() {
			a.openArgFile(arg)
			continue
		}
		a.openProjectPath(arg)
	}
}

// openArgFile shows the file jun was started with in the current project.
func (a *app) openArgFile(path string) {
	p := a.state.CurrentProject()
	if p == nil {
		a.report(fmt.Errorf(a.text("Open a project to show a file in: %s", "需要先打开一个项目才能显示文件：%s"), path))
		return
	}
	if err := a.showFile(p.Name, path); err != nil {
		a.report(err)
	}
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
			ui.Text(c, a.text("Open Project", "打开项目")).FontSize(theme.Title).Bold()
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
