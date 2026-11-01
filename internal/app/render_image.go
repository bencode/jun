package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// imageKind shows pictures, decoded off the UI thread, as large as they
// are and at most as wide as the tab.
var imageKind = renderKind{
	label: func(a *app) string { return a.text("Image", "图片") },
	accepts: func(name string, _ []byte) bool {
		return hasExtension(name, ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg")
	},
	make: func(f fileEnv) fileRender { return &imageRender{a: f.a} },
}

type imageRender struct {
	a      *app
	src    ui.ImageSource
	err    error
	decode int // counts decodes, so that only the latest's result shows
}

func (m *imageRender) update(data []byte) {
	m.decode++
	decode := m.decode
	var src ui.ImageSource
	var err error
	m.a.background(func() { src, err = ui.DecodeImage(data) }, func() {
		if decode == m.decode {
			m.src, m.err = src, err
		}
	})
}

func (m *imageRender) view(c *ui.Context) {
	if m.err != nil {
		ui.Text(c, fmt.Sprintf("%s: %v", m.a.text("Cannot show the image", "无法显示图片"), m.err)).
			TextColor(m.a.theme.Danger).Padding(28)
		return
	}
	if m.src == nil {
		return
	}
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(24).AlignItems(ui.Center).Children(func() {
			img := ui.Image(c, m.src).MaxWidthPercent(100) // as large as it is, at most as wide as the tab
			if w, h := ui.ImageSize(m.src); w > 0 && h > 0 {
				img.AspectRatio(w / h)
			}
		})
	})
}

func (m *imageRender) detail() string {
	if w, h := ui.ImageSize(m.src); w > 0 && h > 0 {
		return fmt.Sprintf("%.0f × %.0f", w, h)
	}
	return ""
}
