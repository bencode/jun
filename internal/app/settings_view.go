package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/settings"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// settingsTab is the settings file as the settings tabs edit it, one
// buffer for them all, and whether the file changed while the buffer had
// changes of its own.
type settingsTab struct {
	editor  *editor.State
	changed bool
}

// openSettings shows the current perspective's settings tab, else opens
// one where new tabs go.
func (a *app) openSettings() {
	project, target := a.state.CurrentProject(), a.tabsTarget()
	if project == nil || target == nil {
		a.report(errors.New(a.text("Open a project first: the settings open in one of its tabs", "请先打开一个项目：设置在项目的标签页中打开")))
		return
	}
	for _, r := range a.perspectiveTabRegions() {
		if i := slices.IndexFunc(r.Views, func(v *layout.View) bool { return v.Kind == layout.ViewSettings }); i >= 0 {
			a.report(a.state.SelectView(r.ID, r.Views[i].ID))
			a.layoutChanged()
			a.focusRegion(r.ID)
			return
		}
	}
	if _, err := a.state.OpenView(target.ID, layout.ViewSettings, project.Name); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(target.ID)
}

// perspectiveTabRegions are the current perspective's columns and bottom
// region.
func (a *app) perspectiveTabRegions() []*layout.Region {
	p := a.state.CurrentPerspective()
	if p == nil {
		return nil
	}
	return append(slices.Clone(p.Columns), p.Bottom)
}

// settingsBuffer is the settings tabs' buffer, reading the file, or
// starting from the template while there is none, the first time.
func (a *app) settingsBuffer() *settingsTab {
	if a.settingsTab == nil {
		text, err := a.settingsFile()
		if err != nil {
			a.report(err)
		}
		buffer := editor.NewBuffer(text)
		buffer.SetLanguage(editor.LanguageFor("settings.yaml"))
		a.settingsTab = &settingsTab{editor: &editor.State{Buffer: buffer, Label: a.settingsPath}}
	}
	return a.settingsTab
}

// settingsFile is the settings file's text, or the template while there
// is no file.
func (a *app) settingsFile() (string, error) {
	data, err := os.ReadFile(a.settingsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return settings.Template, nil
	}
	return string(data), err
}

// settingsFileChanged shows the file as it now is, unless the buffer has
// changes of its own: then it notes that the file changed.
func (a *app) settingsFileChanged() {
	t := a.settingsTab
	if t == nil {
		return
	}
	text, err := a.settingsFile()
	if err != nil || text == t.editor.Buffer.String() {
		return
	}
	if t.editor.Buffer.Modified() {
		t.changed = true
		return
	}
	t.editor.Buffer.Reset(text)
}

// saveSettings writes the buffer to the settings file, and applies it.
func (a *app) saveSettings() {
	t := a.settingsTab
	if t == nil {
		return
	}
	if err := settings.Write(a.settingsPath, []byte(t.editor.Buffer.String())); err != nil {
		a.report(err)
		return
	}
	t.editor.Buffer.MarkSaved()
	t.changed = false
	a.reloadPrefs()
}

// reloadSettingsFile drops the buffer's changes for the file as it is.
func (a *app) reloadSettingsFile() {
	text, err := a.settingsFile()
	if err != nil {
		a.report(err)
		return
	}
	a.settingsTab.editor.Buffer.Reset(text)
	a.settingsTab.changed = false
}

// settingsShortcut saves the settings with ⌘S while a settings tab shows
// in the region that has the focus.
func (a *app) settingsShortcut(c *ui.Context) {
	c.OnShortcut(ui.Super, ui.KeyS, func() {
		if r := a.targetRegion(); r != nil {
			if v := activeView(r); v != nil && v.Kind == layout.ViewSettings {
				a.saveSettings()
			}
		}
	})
}

// settingsView edits the settings file: ⌘S saves it, which applies it.
func (a *app) settingsView(c *ui.Context, v *layout.View) {
	t := a.settingsBuffer()
	modified := t.editor.Buffer.Modified()
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		ui.Row(c).FillWidth().Height(28).Padding(0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.settingsPath).FontSize(theme.Caption).MinWidth(0).SingleLine().Ellipsis("…").Shrink(1)
			if modified {
				ui.Text(c, a.text("● Unsaved", "● 未保存")).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine()
			}
			ui.Spacer(c)
			if t.changed {
				ui.Text(c, a.text("Changed on disk", "文件已在外部修改")).FontSize(theme.Caption).TextColor(a.theme.Warning).SingleLine()
				ui.Button(c, a.text("Reload", "重新加载")).Height(22).FontSize(theme.Caption).OnClick(a.reloadSettingsFile)
			}
			ui.Button(c, a.text("Save", "保存")).Height(22).FontSize(theme.Caption).Disabled(!modified).OnClick(a.saveSettings)
		})
		if a.prefsErr != nil {
			ui.Text(c, fmt.Sprintf("%s: %v", a.text("Not applied", "未生效"), a.prefsErr)).FontSize(theme.Caption).
				TextColor(a.theme.Danger).Padding(2, 12, 4)
		}
		t.editor.Font.Size = a.textSize()
		editor.View(c, t.editor).Grow(1).FillWidth()
	})
}

// closingSettings reports whether closing the view loses unsaved changes:
// it is the last settings tab, and the buffer has them.
func (a *app) closingSettings(view layout.ViewID) (lost bool) {
	if a.settingsTab == nil || a.countSettingsViews(view) > 0 {
		return false
	}
	return a.settingsTab.editor.Buffer.Modified()
}

// countSettingsViews counts the settings tabs but the view.
func (a *app) countSettingsViews(except layout.ViewID) int {
	n := 0
	for _, p := range a.state.Projects {
		for _, perspective := range p.Perspectives {
			for _, r := range append(slices.Clone(perspective.Columns), perspective.Bottom) {
				n += len(slices.DeleteFunc(slices.Clone(r.Views), func(v *layout.View) bool {
					return v.Kind != layout.ViewSettings || v.ID == except
				}))
			}
		}
	}
	return n
}
