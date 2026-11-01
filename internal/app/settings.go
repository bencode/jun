package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *app) settingsButton(c *ui.Context) {
	label := a.text("Settings", "设置")
	a.railAction(ui.ButtonBase(c.Key("settings")).Height(36).FillWidth().Shrink(0).Padding(0, 10).Gap(8).Justify(ui.Start).
		Label(label).Tooltip(label)).Children(func() {
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
	if m.Item(a.text("Font…", "字体…")).Chosen() {
		a.fontOpen = true
	}
	if m.Item(a.text("Theme…", "主题…")).Chosen() {
		a.openThemePalette()
	}
}

func (a *app) setLanguage(chinese bool) {
	a.state.Chinese = chinese
	a.layoutChanged()
	mygo.App.SetMenu(a.applicationMenu())
}

// defaultTextSize is terminals' and file tabs' text size until it is
// changed: the editor's own, and Ghostty's.
const defaultTextSize float32 = 13

// Text sizes ⌘+ and ⌘− zoom between.
const minTextSize, maxTextSize float32 = 8, 40

// textSize is the size of terminals' and file tabs' text.
func (a *app) textSize() float32 {
	if a.state.TextSize > 0 {
		return a.state.TextSize
	}
	return defaultTextSize
}

// setTextSize gives terminals and file tabs text of size DIPs, within
// the limits, and saves it.
func (a *app) setTextSize(size float32) {
	a.state.TextSize = min(max(size, minTextSize), maxTextSize)
	a.terminals.SetTextSize(a.state.TextSize)
	a.layoutChanged()
}
