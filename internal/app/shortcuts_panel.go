package app

import (
	"strings"

	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// openShortcuts shows the Keyboard Shortcuts panel, its search empty.
func (a *app) openShortcuts() {
	a.shortcutsPanel.open, a.shortcutsPanel.query = true, ""
}

// shortcutsDialog lists the shortcuts, by group, matching what is typed in
// its search field. It only shows them: it runs and changes none.
func (a *app) shortcutsDialog(c *ui.Context) {
	s := &a.shortcutsPanel
	done := func() {
		s.open = false
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	}
	modal := ui.Modal(c, &s.open, func() {
		ui.Column(c).Width(560).Gap(10).Children(func() {
			ui.Text(c, a.text("Keyboard Shortcuts", "键盘快捷键")).FontSize(theme.Title).Bold()
			ui.TextInput(c, &s.query).Label(a.text("Search shortcuts", "搜索快捷键")).
				Placeholder(a.text("Search actions or keys, such as close or cmd+w", "搜索动作或按键，如 关闭 或 cmd+w")).AutoFocus().FillWidth()
			ui.Text(c, "⌘ Command   ⌥ Option   ⌃ Control   ⇧ Shift").FontSize(theme.Caption).TextColor(a.theme.TextMuted)
			groups := matchShortcuts(shortcutGroups(), s.query)
			if len(groups) == 0 {
				ui.Text(c, a.text("No shortcuts match", "没有匹配的快捷键")).TextColor(a.theme.TextMuted).Padding(12, 0)
			} else {
				ui.Scroll(c).MaxHeight(420).FillWidth().Children(func() {
					ui.Column(c).FillWidth().Gap(14).Padding(0, 4, 0, 0).Children(func() {
						for _, g := range groups {
							a.shortcutGroupView(c.Key(g.name.en), g)
						}
					})
				})
			}
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				ui.PrimaryButton(c, a.text("Done", "完成")).OnClick(done)
			})
		})
	})
	if modal.Dismissed() {
		done()
	}
}

func (a *app) shortcutGroupView(c *ui.Context, g shortcutGroup) {
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, a.say(g.name)).FontSize(theme.Caption).Bold().TextColor(a.theme.TextMuted)
		for _, r := range g.rows {
			ui.Row(c.Key(r.action.en)).FillWidth().Gap(16).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, a.say(r.action))
					if r.note.en != "" {
						ui.Text(c, a.say(r.note)).FontSize(theme.Caption).TextColor(a.theme.TextMuted)
					}
				})
				ui.Text(c, strings.Join(r.keys, " / ")).Shrink(0)
			})
		}
	})
}
