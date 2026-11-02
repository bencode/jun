package app

import (
	"slices"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/egoist/mygo/ui"
)

// lastTab is the index ⌘9 selects, following browsers and Ghostty.
const lastTab = -1

var digitKeys = []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}

// keyCombo is a shortcut: key, a letter or digit, pressed with exactly
// mods. Those shown in more than one place are named here, so that the
// bindings, menus, tooltips, the palette and the Keyboard Shortcuts panel
// agree.
type keyCombo struct {
	mods ui.Modifiers
	key  ui.Key
}

var (
	commandPaletteKeys = keyCombo{ui.Super, ui.KeyK}
	reviewChangesKeys  = keyCombo{ui.Super | ui.Shift, ui.KeyD}
	openProjectKeys    = keyCombo{ui.Super, ui.KeyO}
	newTerminalKeys    = keyCombo{ui.Super, ui.KeyT}
	closeTabKeys       = keyCombo{ui.Super, ui.KeyW}
	toggleSideKeys     = keyCombo{ui.Super, ui.KeyB}
	newPerspectiveKeys = keyCombo{ui.Super | ui.Shift, ui.KeyN}
)

// perspectiveKeys and projectKeys select the perspective or project at
// index; only the first nine have keys.
func perspectiveKeys(index int) (keyCombo, bool) { return digitCombo(ui.Super|ui.Alt, index) }
func projectKeys(index int) (keyCombo, bool)     { return digitCombo(ui.Ctrl|ui.Super, index) }

func digitCombo(mods ui.Modifiers, index int) (keyCombo, bool) {
	if index < 0 || index >= len(digitKeys) {
		return keyCombo{}, false
	}
	return keyCombo{mods, digitKeys[index]}, true
}

// modifierNames are the modifiers in the order shortcuts show them, with
// their native menus' names.
var modifierNames = []struct {
	mod          ui.Modifiers
	symbol, name string
}{{ui.Ctrl, "⌃", "Ctrl"}, {ui.Super, "⌘", "CmdOrCtrl"}, {ui.Alt, "⌥", "Alt"}, {ui.Shift, "⇧", "Shift"}}

// String shows the combo as menus do, such as ⌘⇧N.
func (k keyCombo) String() string {
	var b strings.Builder
	for _, m := range modifierNames {
		if k.mods&m.mod != 0 {
			b.WriteString(m.symbol)
		}
	}
	return b.String() + k.keyName()
}

// accelerator is the combo for a native menu item, such as
// CmdOrCtrl+Shift+N.
func (k keyCombo) accelerator() string {
	var parts []string
	for _, m := range modifierNames {
		if k.mods&m.mod != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, k.keyName()), "+")
}

func (k keyCombo) keyName() string {
	if k.key >= ui.Key0 && k.key <= ui.Key9 {
		return string(rune('0' + k.key - ui.Key0))
	}
	return string(rune('A' + k.key - ui.KeyA))
}

func (a *app) shortcuts(c *ui.Context) {
	c.OnShortcut(newTerminalKeys.mods, newTerminalKeys.key, func() {
		if r := a.tabsTarget(); r != nil {
			a.newTerminal(r.ID)
		}
	})
	c.OnShortcut(closeTabKeys.mods, closeTabKeys.key, func() {
		if r := a.targetRegion(); r != nil && r.Active != 0 && !a.state.Fixed(r.ID) {
			a.closeView(r.ID, r.Active)
		}
	})
	// ⌘B, ⌘K, ⌘O and ⌘⇧D belong to the menu bar (applicationMenu), which
	// takes them before the view, dialogs open or not.
	c.OnShortcut(ui.Super, ui.KeyP, a.searchFiles)
	c.OnShortcut(ui.Super, ui.KeyEqual, func() { a.setTextSize(a.textSize() + 1) })
	c.OnShortcut(ui.Super|ui.Shift, ui.KeyEqual, func() { a.setTextSize(a.textSize() + 1) })
	c.OnShortcut(ui.Super, ui.KeyMinus, func() { a.setTextSize(a.textSize() - 1) })
	c.OnShortcut(ui.Super, ui.Key0, func() { a.setTextSize(defaultTextSize) })
	c.OnShortcut(ui.Ctrl, ui.KeyTab, func() { a.cycleTab(1) })
	c.OnShortcut(ui.Ctrl|ui.Shift, ui.KeyTab, func() { a.cycleTab(-1) })
	c.OnShortcut(ui.Super|ui.Shift, ui.KeyBracketRight, func() { a.cycleTab(1) })
	c.OnShortcut(ui.Super|ui.Shift, ui.KeyBracketLeft, func() { a.cycleTab(-1) })
	c.OnShortcut(ui.Super, ui.KeyBracketRight, func() { a.cycleRegion(1) })
	c.OnShortcut(ui.Super, ui.KeyBracketLeft, func() { a.cycleRegion(-1) })
	c.OnShortcut(ui.Super|ui.Alt, ui.KeyLeft, func() { a.focusDirection(focusLeft) })
	c.OnShortcut(ui.Super|ui.Alt, ui.KeyRight, func() { a.focusDirection(focusRight) })
	c.OnShortcut(ui.Super|ui.Alt, ui.KeyUp, func() { a.focusDirection(focusUp) })
	c.OnShortcut(ui.Super|ui.Alt, ui.KeyDown, func() { a.focusDirection(focusDown) })
	c.OnShortcut(ui.Super|ui.Alt, ui.KeyBracketRight, func() { a.cyclePerspective(1) })
	c.OnShortcut(ui.Super|ui.Alt, ui.KeyBracketLeft, func() { a.cyclePerspective(-1) })
	c.OnShortcut(newPerspectiveKeys.mods, newPerspectiveKeys.key, func() { a.openPerspectiveForm(0) })
	c.OnShortcut(ui.Ctrl|ui.Super, ui.KeyBracketRight, func() { a.cycleProject(1) })
	c.OnShortcut(ui.Ctrl|ui.Super, ui.KeyBracketLeft, func() { a.cycleProject(-1) })
	for i, key := range digitKeys {
		tab := i
		if i == len(digitKeys)-1 {
			tab = lastTab
		}
		c.OnShortcut(ui.Super, key, func() { a.selectTabAt(tab) })
		perspective, _ := perspectiveKeys(i)
		project, _ := projectKeys(i)
		c.OnShortcut(perspective.mods, perspective.key, func() { a.selectPerspectiveAt(i) })
		c.OnShortcut(project.mods, project.key, func() { a.selectProjectAt(i) })
	}
}

func (a *app) selectTabAt(index int) {
	r := a.targetRegion()
	if r == nil || len(r.Views) == 0 {
		return
	}
	if index == lastTab {
		index = len(r.Views) - 1
	}
	if index < len(r.Views) {
		a.selectTab(r, r.Views[index].ID)
	}
}

func (a *app) cyclePerspective(delta int) {
	project := a.state.CurrentProject()
	if project == nil {
		return
	}
	index := slices.IndexFunc(project.Perspectives, func(p *layout.Perspective) bool { return p.ID == project.ActivePerspective })
	n := len(project.Perspectives)
	a.selectPerspective(project.Perspectives[(index+delta+n)%n].ID)
}

func (a *app) selectPerspectiveAt(index int) {
	if project := a.state.CurrentProject(); project != nil && index < len(project.Perspectives) {
		a.selectPerspective(project.Perspectives[index].ID)
	}
}

func (a *app) cycleProject(delta int) {
	n := len(a.state.Projects)
	if n == 0 {
		return
	}
	index := slices.IndexFunc(a.state.Projects, func(p *layout.Project) bool { return p.Name == a.state.ActiveProject })
	a.selectProject(a.state.Projects[(index+delta+n)%n].Name)
}

func (a *app) selectProjectAt(index int) {
	if index < len(a.state.Projects) {
		a.selectProject(a.state.Projects[index].Name)
	}
}
