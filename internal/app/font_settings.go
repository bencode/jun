package app

import (
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// fontDialog is the Font panel, for the text of terminals and file tabs.
// Each setting applies and is saved as it changes, so it only closes.
func (a *app) fontDialog(c *ui.Context) {
	done := func() {
		a.fontOpen = false
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	}
	modal := ui.Modal(c, &a.fontOpen, func() {
		ui.Column(c).Width(380).Gap(12).Children(func() {
			ui.Text(c, a.text("Font", "字体")).FontSize(theme.Title).Bold()
			ui.Text(c, a.text("For terminals and file tabs.", "用于终端和文件标签。")).TextColor(a.theme.TextMuted)
			ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.text("Size", "字号")).Width(64).Shrink(0)
				size := float64(a.textSize())
				if ui.NumberInput(c, &size, float64(minTextSize), float64(maxTextSize), 1).Label(a.text("Text size", "文字大小")).Width(110).Changed() {
					a.setTextSize(float32(size))
				}
				ui.Spacer(c)
				ui.Button(c, a.text("Reset", "重置")).OnClick(func() { a.setTextSize(defaultTextSize) })
			})
			ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
				ui.PrimaryButton(c, a.text("Done", "完成")).OnClick(done)
			})
		})
	})
	if modal.Dismissed() {
		done()
	}
}
