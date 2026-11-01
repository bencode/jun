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

// Default is Jun's own theme; a layout without a theme uses it.
const Default = "Jun"

// Names lists the themes to choose from: Jun's, then Ghostty's.
func Names() []string { return append([]string{Default}, ghostty.GhosttyThemes()...) }

// Terminal returns the named theme's terminal colors.
func Terminal(name string) (*ghostty.Theme, error) {
	if name != "" && name != Default {
		return ghostty.GhosttyTheme(name)
	}
	return JunTerminal(), nil
}

// JunTerminal is Jun's own terminal colors: Ghostty's dark theme on Jun's
// background and text.
func JunTerminal() *ghostty.Theme {
	t, u := ghostty.DarkTheme(), jun()
	t.Foreground, t.Background = u.Text, u.Background
	return t
}

// UI returns the controls' colors for the named theme: Jun's own, or ones
// drawn from the terminal's background and text.
func UI(name string, term *ghostty.Theme) *ui.Theme {
	if name == "" || name == Default {
		return jun()
	}
	return derive(term)
}

// IsDark reports whether a theme's background is dark.
func IsDark(t *ui.Theme) bool { return luminance(t.Background) <= 0.5 }

// jun is Jun's dark theme: MyGo's, in the cool blue-grays of the logo.
func jun() *ui.Theme {
	t := ui.DarkTheme()
	t.Background, t.Surface = ui.Hex("#15181e"), ui.Hex("#1d2129")
	t.SurfaceHover, t.SurfacePressed = ui.Hex("#262b35"), ui.Hex("#2e3440")
	t.Border = ui.Hex("#2f3541")
	t.Text, t.TextMuted = ui.Hex("#e8ecf3"), ui.Hex("#8b93a3")
	t.Accent, t.AccentHover, t.AccentPressed = ui.Hex("#3f6fe0"), ui.Hex("#5584f0"), ui.Hex("#3461cc")
	t.Selection, t.Focus = ui.RGBA(122, 162, 255, 0.28), ui.RGBA(122, 162, 255, 0.6)
	return t
}

// derive draws controls from a terminal's colors: surfaces and borders
// steps from its background toward its text, on MyGo's light or dark
// theme for the status colors, and Jun's blue, which reads on either.
func derive(term *ghostty.Theme) *ui.Theme {
	bg, fg := term.Background, term.Foreground
	t := ui.DarkTheme()
	if luminance(bg) > 0.5 {
		t = ui.LightTheme()
	}
	t.Background, t.Text = bg, fg
	t.Surface, t.SurfaceHover, t.SurfacePressed = bg.Mix(fg, 0.06), bg.Mix(fg, 0.10), bg.Mix(fg, 0.14)
	t.Border, t.TextMuted = bg.Mix(fg, 0.14), bg.Mix(fg, 0.55)
	j := jun()
	t.Accent, t.AccentHover, t.AccentPressed = j.Accent, j.AccentHover, j.AccentPressed
	t.Selection, t.Focus = j.Selection, j.Focus
	return t
}

// luminance is a color's relative brightness, from 0 to 1.
func luminance(c ui.Color) float32 {
	return (0.2126*float32(c.R) + 0.7152*float32(c.G) + 0.0722*float32(c.B)) / 255
}
