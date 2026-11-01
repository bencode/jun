// Package theme holds Jun's colors, for its controls and its terminals.
package theme

import (
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// Text sizes besides the theme's body size: captions, dialog titles and
// headings.
const (
	Caption float32 = 11
	Title   float32 = 16
	Heading float32 = 22
)

// UI is Jun's dark theme: MyGo's, in the cool blue-grays of the logo.
func UI() *ui.Theme {
	t := ui.DarkTheme()
	t.Background, t.Surface = ui.Hex("#15181e"), ui.Hex("#1d2129")
	t.SurfaceHover, t.SurfacePressed = ui.Hex("#262b35"), ui.Hex("#2e3440")
	t.Border = ui.Hex("#2f3541")
	t.Text, t.TextMuted = ui.Hex("#e8ecf3"), ui.Hex("#8b93a3")
	t.Accent, t.AccentHover, t.AccentPressed = ui.Hex("#3f6fe0"), ui.Hex("#5584f0"), ui.Hex("#3461cc")
	t.Selection, t.Focus = ui.RGBA(122, 162, 255, 0.28), ui.RGBA(122, 162, 255, 0.6)
	return t
}

// Terminal is Ghostty's dark theme on UI's background and text.
func Terminal() *ghostty.Theme {
	t, u := ghostty.DarkTheme(), UI()
	t.Foreground, t.Background = u.Text, u.Background
	return t
}
