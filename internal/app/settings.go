package app

import (
	"cmp"

	"github.com/bencode/jun/internal/settings"
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
		if m.Item("English").Radio(!a.chinese()).Chosen() {
			a.setLanguage(false)
		}
		if m.Item("简体中文").Radio(a.chinese()).Chosen() {
			a.setLanguage(true)
		}
	})
	if m.Item(a.text("Theme…", "主题…")).Chosen() {
		a.openThemePalette()
	}
	m.Separator()
	if m.Item(a.text("Open Settings File", "打开设置文件")).Shortcut(ui.Super, ui.KeyComma).Disabled(a.state.CurrentProject() == nil).Chosen() {
		a.openSettings()
	}
}

// chinese reports whether the interface is in Chinese.
func (a *app) chinese() bool { return a.prefs.Language == "zh" }

// setLanguage shows the interface in Chinese or English, and saves it.
func (a *app) setLanguage(chinese bool) {
	language := "en"
	if chinese {
		language = "zh"
	}
	a.setPref([]string{"language"}, language)
}

// defaultTextSize is terminals' and file tabs' text size until it is
// set: the editor's own, and Ghostty's.
const defaultTextSize = settings.DefaultFontSize

// textSize is the size of terminals' and file tabs' text.
func (a *app) textSize() float32 {
	return cmp.Or(a.prefs.Font.Size, defaultTextSize)
}

// setTextSize gives terminals and file tabs text of size DIPs, within
// the limits, and saves it.
func (a *app) setTextSize(size float32) {
	a.setPref([]string{"font", "size"}, min(max(size, settings.MinFontSize), settings.MaxFontSize))
}
