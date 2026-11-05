package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/bencode/jun/internal/message"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// messagesView lists what agents told the user, the latest first.
func (a *app) messagesView(c *ui.Context) {
	c.After(time.Minute) // for the ages to follow
	ui.Column(c.Key("messages")).Fill().Children(func() {
		if len(a.messages) == 0 {
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Padding(12).Children(func() {
				ui.Text(c, a.text("Agents tell you here, with jun tool message.", "agent 用 jun tool message 发到这里。")).
					TextColor(a.theme.TextMuted)
			})
			return
		}
		ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).TrackScroll(&a.messagesScroll).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Padding(0, 4, 8, 4).Children(func() {
				now := time.Now()
				for _, m := range slices.Backward(a.messages) {
					a.messageRow(c.Key(m.Time.UnixNano()), m, now)
				}
			})
		})
	})
}

func (a *app) messageRow(c *ui.Context, m message.Message, now time.Time) {
	content := func() {
		ui.Text(c, fmt.Sprintf("#%d %s · %s", m.Terminal, m.Project, a.messageAge(now, m.Time))).
			FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…")
		ui.Text(c, m.Text)
	}
	id, running := a.messageTerminal(m)
	if !running {
		ui.Column(c).FillWidth().Gap(2).Padding(6, 8).Border(2, ui.Transparent).Tooltip(a.text("The terminal has ended", "这个终端已结束")).Children(content)
		return
	}
	border := ui.Transparent // always there, for a peek to move no row
	if id == a.peek.id {
		border = a.theme.Accent
	}
	row := ui.ButtonBase(c).FillWidth().Column().AlignItems(ui.Stretch).Gap(2).Padding(6, 8).Radius(6).Border(2, border).Label(m.Text).Children(content)
	row.OnClick(a.openMessage(row, id))
}

// messageAge tells how long ago a message came, or when after a day.
func (a *app) messageAge(now, at time.Time) string {
	age := now.Sub(at)
	if age < time.Minute {
		return a.text("just now", "刚刚")
	}
	if age < time.Hour {
		return fmt.Sprintf(a.text("%d min ago", "%d 分钟前"), int(age.Minutes()))
	}
	if age < 24*time.Hour {
		return fmt.Sprintf(a.text("%d h ago", "%d 小时前"), int(age.Hours()))
	}
	return at.Local().Format("01-02 15:04")
}
