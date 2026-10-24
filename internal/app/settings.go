package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) settingsButton(c *ui.Context) {
	label := a.text("Settings", "设置")
	ui.ButtonBase(c.Key("settings")).Height(36).FillWidth().Shrink(0).Padding(0, 10).Gap(8).Justify(ui.Start).
		Label(label).Tooltip(label).Children(func() {
		ui.Icon(c, settingsIcon).Size(18, 18).Shrink(0)
		if a.state.RailExpanded {
			ui.Text(c, label).SingleLine()
		}
	}).Menu(a.settingsMenu)
}

func (a *app) settingsMenu(m *ui.Menu) {
	if m.Item("English").Checked(!a.chinese).Chosen() {
		a.setLanguage(false)
	}
	if m.Item("简体中文").Checked(a.chinese).Chosen() {
		a.setLanguage(true)
	}
}

func (a *app) setLanguage(chinese bool) {
	a.chinese = chinese
	mygo.App.SetMenu(a.applicationMenu())
}
