package app

import (
	"fmt"
	"slices"

	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
)

// applyTheme colors Jun with the named theme, without saving it.
func (a *app) applyTheme(name string) error {
	term, err := theme.Terminal(name)
	if err != nil {
		return err
	}
	a.theme = theme.UI(name, term)
	a.terminals.SetTheme(term)
	for _, p := range a.previews {
		p.term.SetTheme(term, nil)
	}
	a.matchAppearance()
	return nil
}

// matchAppearance gives the window's native parts, such as its title bar,
// the theme's lightness, once there is a window.
func (a *app) matchAppearance() {
	if a.window == nil {
		return
	}
	source := mygo.ThemeLight
	if theme.IsDark(a.theme) {
		source = mygo.ThemeDark
	}
	mygo.Theme.SetSource(source)
}

// useSavedTheme applies the layout's theme, or Jun's, saying why, when
// the layout's is gone.
func (a *app) useSavedTheme() {
	err := a.applyTheme(a.state.Theme)
	if err == nil {
		return
	}
	a.report(fmt.Errorf("%s: %w", a.text("Theme unavailable; using Jun's", "主题不可用，改用 Jun 的主题"), err))
	if err := a.applyTheme(theme.Default); err != nil {
		a.report(err)
	}
}

// openThemePalette lists the themes in the palette, at the current one,
// which each highlighted theme replaces until one is chosen.
func (a *app) openThemePalette() {
	current := a.state.Theme
	if current == "" {
		current = theme.Default
	}
	names := theme.Names()
	p := &a.palette
	p.open, p.query, p.caretEnd = true, "", false
	p.themes, p.previewed, p.reveal = names, current, true
	p.highlight = max(slices.Index(names, current), 0)
}

// leaveThemes ends choosing a theme with none chosen: the saved theme
// shows again.
func (a *app) leaveThemes() {
	if a.palette.themes == nil {
		return
	}
	a.palette.themes = nil
	if err := a.applyTheme(a.state.Theme); err != nil {
		a.report(err)
	}
}

// chooseTheme shows the theme and saves it, Jun's as none.
func (a *app) chooseTheme(name string) {
	a.palette.themes = nil
	if err := a.applyTheme(name); err != nil {
		a.report(err)
		if err := a.applyTheme(a.state.Theme); err != nil {
			a.report(err)
		}
		return
	}
	if name == theme.Default {
		name = ""
	}
	a.state.Theme = name
	a.layoutChanged()
}
