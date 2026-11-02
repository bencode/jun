package app

import (
	"fmt"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestShortcutSearch(t *testing.T) {
	actions := func(query string) []string {
		var names []string
		for _, g := range matchShortcuts(shortcutGroups(), query) {
			if len(g.rows) == 0 {
				t.Errorf("%q kept the empty group %s", query, g.name.en)
			}
			for _, r := range g.rows {
				names = append(names, r.action.en)
			}
		}
		return names
	}
	for query, want := range map[string]string{
		"关闭": "Close tab", "close": "Close tab", "cmd+w": "Close tab", " CMD+W ": "Close tab", "⌘W": "Close tab",
		"shift+cmd+n": "New perspective", "ctrl+shift+tab": "Previous tab", "cmd++": "Larger text", "cmd+-": "Smaller text",
	} {
		if !slices.Contains(actions(query), want) {
			t.Errorf("searching %q did not find %s", query, want)
		}
	}
	if got := actions("no such shortcut"); got != nil {
		t.Errorf("an unmatched search found %q", got)
	}
	if len(actions("")) == 0 {
		t.Error("an empty search lists nothing")
	}
}

func TestShortcutsPanelEntries(t *testing.T) {
	a, tt := testApp(t)
	right := a.state.CurrentPerspective().Columns[1]
	a.focusRegion(right.ID)
	tt.Frame()
	views := len(right.Views)
	open := func(how string, open func()) {
		open()
		tt.Frame()
		if !a.shortcutsPanel.open || a.palette.open {
			t.Fatalf("%s: panel open %v, palette open %v", how, a.shortcutsPanel.open, a.palette.open)
		}
		tt.Type("close")
		tt.Frame()
		if a.shortcutsPanel.query != "close" || !tt.HasText("Close tab") || tt.HasText("New terminal") {
			t.Fatalf("%s: typing did not search: %q", how, a.shortcutsPanel.query)
		}
		tt.Key(ui.Super, ui.KeyT)
		tt.Key(0, ui.KeyEscape)
		tt.Frame()
		if a.shortcutsPanel.open || len(right.Views) != views || !tt.Focused(contentLabel(a, right.Views[0])) {
			t.Fatalf("%s: Escape left the panel open, ⌘T reached the region, or focus did not return", how)
		}
	}
	open("Settings", func() {
		click(t, tt, "Settings")
		if err := tt.ChooseMenuItem("Keyboard Shortcuts…"); err != nil {
			t.Fatal(err)
		}
	})
	open("palette", func() {
		a.openPalette("")
		tt.Frame()
		tt.Type("keys")
		tt.Key(0, ui.KeyEnter)
	})
}

func TestPaletteShowsBoundKeys(t *testing.T) {
	a := testNewApp(t)
	for i := 1; i <= 10; i++ {
		if err := a.state.AddProject(fmt.Sprintf("p%d", i), t.TempDir()); err != nil {
			t.Fatal(err)
		}
	}
	keys := map[string]string{}
	for _, item := range a.commands() {
		keys[item.title] = item.keys
	}
	for title, want := range map[string]string{
		"New Terminal": "⌘T", "Toggle Side Bar": "⌘B", "Theme…": "", "Project: p9": "⌃⌘9", "Project: p10": "", "Perspective: Work": "⌘⌥1",
	} {
		if got, ok := keys[title]; !ok || got != want {
			t.Errorf("%s shows keys %q, want %q (listed %v)", title, got, want, ok)
		}
	}
}
