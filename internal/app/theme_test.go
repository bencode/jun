package app

import (
	"testing"

	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

func TestThemePalettePreviewsChoosesAndReverts(t *testing.T) {
	a, tt := testApp(t)
	black, jun := ui.Hex("#000000"), a.theme.Background
	a.openThemePalette()
	tt.Frame()
	tt.Type("Wez")
	tt.Frame()
	if a.theme.Background != black || a.terminals.Theme().Background != black || a.state.Theme != "" {
		t.Fatal("highlighting Wez did not preview it, or saved it")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.palette.open || a.theme.Background != jun || a.terminals.Theme().Background != jun {
		t.Fatal("closing the palette did not bring back Jun's theme")
	}
	a.openThemePalette()
	tt.Frame()
	tt.Type("Wez")
	tt.Frame()
	a.openPalette("") // ⌘K, which menus pass on with a dialog open
	tt.Frame()
	if a.theme.Background != jun {
		t.Fatal("opening the palette for commands kept an unchosen theme")
	}
	a.openThemePalette()
	tt.Frame()
	tt.Type("Wez")
	tt.Key(0, ui.KeyEnter)
	if a.state.Theme != "Wez" || a.theme.Background != black || a.terminals.Theme().Background != black {
		t.Fatal("Enter did not choose and save Wez")
	}
	a.state.Theme = "No Such Theme"
	a.useSavedTheme()
	if !a.errorOpen || a.theme.Background != theme.UI(theme.Default, nil).Background {
		t.Fatal("a theme gone did not fall back to Jun's, saying why")
	}
}
