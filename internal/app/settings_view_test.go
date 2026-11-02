package app

import (
	"os"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/settings"
	"github.com/egoist/mygo/ui"
)

func TestSettingsTabEditsSavesAndAppliesTheFile(t *testing.T) {
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1200, 700)
	tt.Frame()
	a.openSettings()
	a.openSettings() // shows the one already open
	tt.Frame()
	column := a.state.CurrentPerspective().Columns[0]
	if v := activeView(column); v == nil || v.Kind != layout.ViewSettings || a.countSettingsViews(0) != 1 {
		t.Fatalf("⌘, showed %+v", v)
	}
	buffer := a.settingsTab.editor.Buffer
	if buffer.String() != settings.Template {
		t.Fatalf("with no file the tab offers %q", buffer.String())
	}

	buffer.Set("font:\n  size: 18\n")
	tt.Frame()
	if !tt.HasText("● Unsaved") {
		t.Fatalf("an edit shows %v", tt.Texts())
	}
	tt.Key(ui.Super, ui.KeyS)
	tt.Frame()
	if data, err := os.ReadFile(a.settingsPath); err != nil || string(data) != "font:\n  size: 18\n" || a.textSize() != 18 || buffer.Modified() {
		t.Fatalf("⌘S saved %q (%v), size %v", data, err, a.textSize())
	}

	buffer.Set("font:\n  size: 99\n")
	a.saveSettings()
	tt.Frame()
	if a.textSize() != 18 || a.prefsErr == nil || !tt.HasText("Not applied: "+a.prefsErr.Error()) {
		t.Fatalf("a wrong file applied size %v, %v", a.textSize(), a.prefsErr)
	}

	// The file changing: followed while the buffer has no changes, noted
	// while it has.
	if err := os.WriteFile(a.settingsPath, []byte("language: zh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.reloadPrefs()
	if buffer.String() != "language: zh\n" || !a.chinese() || a.prefsErr != nil {
		t.Fatalf("the tab shows %q after the file changed", buffer.String())
	}
	buffer.Set("language: en\n")
	if err := os.WriteFile(a.settingsPath, []byte("theme: Wez\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.reloadPrefs()
	tt.Frame()
	if buffer.String() != "language: en\n" || !a.settingsTab.changed || !tt.HasText("Changed on disk") {
		t.Fatalf("the tab's changes gave way to the file: %q", buffer.String())
	}

	// Closing the last settings tab with changes asks first.
	a.closeView(column.ID, column.Active)
	if !a.confirmation.open || a.countSettingsViews(0) != 1 {
		t.Fatal("closing unsaved settings did not ask")
	}
	a.confirmation.run()
	if a.countSettingsViews(0) != 0 || a.settingsTab != nil {
		t.Fatal("discarding did not close the tab and forget its changes")
	}
}

func TestMigratePrefsMovesTheLayoutsSettingsToTheFile(t *testing.T) {
	a := testNewApp(t)
	if err := os.WriteFile(a.settingsPath, []byte("# mine\ntheme: Wez\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !a.readPrefs() {
		t.Fatal(a.prefsErr)
	}
	a.state.Chinese, a.state.Theme, a.state.TextSize = true, "Dracula", 15
	a.migratePrefs()
	data, err := os.ReadFile(a.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !a.chinese() || a.prefs.Theme != "Wez" || a.textSize() != 15 || !strings.HasPrefix(string(data), "# mine\n") {
		t.Fatalf("migrated to %q", data)
	}
	if a.state.Chinese || a.state.Theme != "" || a.state.TextSize != 0 {
		t.Fatal("the layout kept the settings it gave the file")
	}
}
