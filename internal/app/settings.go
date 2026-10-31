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
	m.Submenu(a.text("Language", "语言"), func(m *ui.Menu) {
		if m.Item("English").Radio(!a.state.Chinese).Chosen() {
			a.setLanguage(false)
		}
		if m.Item("简体中文").Radio(a.state.Chinese).Chosen() {
			a.setLanguage(true)
		}
	})
}

func (a *app) setLanguage(chinese bool) {
	a.state.Chinese = chinese
	a.layoutChanged()
	mygo.App.SetMenu(a.applicationMenu())
}
