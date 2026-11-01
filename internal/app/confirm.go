package app

import (
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// confirmation asks before an action that cannot be undone.
type confirmation struct {
	open                  bool
	title, detail, action string
	run                   func()
}

func (a *app) confirm(title, detail, action string, run func()) {
	a.confirmation = confirmation{open: true, title: title, detail: detail, action: action, run: run}
}

func (a *app) confirmDialog(c *ui.Context) {
	f := &a.confirmation
	cancel := func() {
		f.open = false
		if r := a.targetRegion(); r != nil {
			a.focusRegion(r.ID)
		}
	}
	modal := ui.Modal(c, &f.open, func() {
		ui.Column(c).Width(400).Gap(12).Children(func() {
			ui.Text(c, f.title).FontSize(theme.Title).Bold()
			ui.Text(c, f.detail).TextColor(a.theme.TextMuted)
			ui.Row(c).Gap(8).Justify(ui.End).FillWidth().Children(func() {
				ui.Button(c, a.text("Cancel", "取消")).OnClick(cancel)
				ui.PrimaryButton(c, f.action).OnClick(func() {
					f.open = false
					f.run()
				})
			})
		})
	})
	if modal.Dismissed() {
		cancel()
	}
}
