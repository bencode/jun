package app

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

func (a *app) openPerspectiveForm(id layout.PerspectiveID) {
	project := a.state.CurrentProject()
	if project == nil {
		return
	}
	name := suggestedPerspectiveName(project)
	if id != 0 {
		p := project.Perspective(id)
		if p == nil {
			return
		}
		name = p.Name
	}
	f := &a.perspectiveForm
	f.open, f.project, f.id, f.name, f.error, f.selectName = true, project.Name, id, name, "", true
}

func (a *app) perspectiveDialog(c *ui.Context) {
	f := &a.perspectiveForm
	cancel := func() {
		f.open = false
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	}
	modal := ui.Modal(c, &f.open, func() {
		title, action := a.text("New Perspective", "新建视角"), a.text("Create", "创建")
		if f.id != 0 {
			title, action = a.text("Rename Perspective", "重命名视角"), a.text("Rename", "重命名")
		}
		ui.Column(c).Width(400).Gap(12).Children(func() {
			ui.Text(c, title).FontSize(theme.Title).Bold()
			input := ui.TextInput(c, &f.name).Label(a.text("Perspective name", "视角名称")).AutoFocus().FillWidth().OnSubmit(a.submitPerspective)
			if f.selectName {
				input.SetTextSelection(0, utf8.RuneCountInString(f.name))
				f.selectName = false
			}
			if f.error != "" {
				ui.Text(c, f.error).TextColor(a.theme.Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).FillWidth().Children(func() {
				ui.Button(c, a.text("Cancel", "取消")).OnClick(cancel)
				ui.PrimaryButton(c, action).OnClick(a.submitPerspective)
			})
		})
	})
	if modal.Dismissed() {
		cancel()
	}
}

func (a *app) submitPerspective() {
	f := &a.perspectiveForm
	project := a.state.CurrentProject()
	if project == nil || project.Name != f.project || f.id != 0 && project.Perspective(f.id) == nil {
		f.error = a.text("The project or perspective is no longer available.", "项目或视角已不可用。")
		return
	}
	name := strings.TrimSpace(f.name)
	if name == "" {
		f.error = a.text("Enter a perspective name.", "请输入视角名称。")
		return
	}
	for _, p := range project.Perspectives {
		if p.ID != f.id && strings.EqualFold(p.Name, name) {
			f.error = a.text("That perspective name already exists.", "视角名称已存在。")
			return
		}
	}
	if f.id == 0 {
		if _, err := a.state.AddPerspective(name); err != nil {
			f.error = err.Error()
			return
		}
		a.layoutChanged()
		a.focusCurrentPerspective()
	} else {
		changed := project.Perspective(f.id).Name != name
		if err := a.state.RenamePerspective(f.id, name); err != nil {
			f.error = err.Error()
			return
		}
		if changed {
			a.layoutChanged()
		}
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	}
	f.open, f.error = false, ""
}

// perspectiveNames are offered, in order, for a new perspective: its name
// is the first one the project lacks, the field holding it selected.
var perspectiveNames = []string{"Work", "Review", "Monitor", "Debug", "Notes"}

func suggestedPerspectiveName(project *layout.Project) string {
	taken := func(name string) bool {
		return slices.ContainsFunc(project.Perspectives, func(p *layout.Perspective) bool { return strings.EqualFold(p.Name, name) })
	}
	if i := slices.IndexFunc(perspectiveNames, func(name string) bool { return !taken(name) }); i >= 0 {
		return perspectiveNames[i]
	}
	for n := len(project.Perspectives) + 1; ; n++ {
		if name := fmt.Sprintf("Perspective %d", n); !taken(name) {
			return name
		}
	}
}
