package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// repliesShown is how many replies a message shows, the latest, until it is
// opened.
const repliesShown = 3

// shownReplies are the replies the message shows now.
func shownReplies(m message) []tool.Reply {
	if m.Whole {
		return m.Replies
	}
	return m.Replies[max(0, len(m.Replies)-repliesShown):]
}

// replyWidth is the room a reply's text has beside its age, in a row of the
// width, past the indent and the line.
func (a *app) replyWidth(c *ui.Context, width float32, age string) float32 {
	measured, _ := c.MeasureText(0, ui.Span{Text: age, Size: a.messageCaption()})
	return width - a.messageIconSize() - 2*messageGap - measured
}

// repliesFold tells whether the message has replies to open.
func (a *app) repliesFold(c *ui.Context, m message, width float32, now time.Time) bool {
	if len(m.Replies) > repliesShown {
		return true
	}
	return slices.ContainsFunc(m.Replies, func(r tool.Reply) bool {
		return messageFolds(c, r.Text, a.replyWidth(c, width, a.messageAge(now, r.Time)), a.textSize())
	})
}

// messageReplies shows the replies under the message's text, in line with
// its title, along a line from its icon.
func (a *app) messageReplies(c *ui.Context, m message, now time.Time, chosen bool) {
	if len(m.Replies) == 0 {
		return
	}
	icon, caption, muted := a.messageIconSize(), a.messageCaption(), a.messageMuted(chosen)
	shown := shownReplies(m)
	ui.Row(c).FillWidth().Gap(icon/2+messageGap-1).AlignItems(ui.Stretch).Padding(0, 0, 0, icon/2).Children(func() {
		ui.Box(c).Width(1).Shrink(0).Background(a.theme.Border)
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			if hidden := len(m.Replies) - len(shown); hidden > 0 {
				ui.Text(c, fmt.Sprintf(a.text("%d earlier", "还有 %d 条"), hidden)).FontSize(caption).TextColor(muted)
			}
			for _, r := range shown {
				a.messageReply(c, r, m.Whole, now, caption, muted)
			}
		})
	})
}

func (a *app) messageReply(c *ui.Context, r tool.Reply, whole bool, now time.Time, caption float32, muted ui.Color) {
	ui.Row(c).FillWidth().Gap(messageGap).AlignItems(ui.Start).Children(func() {
		text := ui.Text(c, r.Text).FontSize(a.textSize()).LineHeight(1.4).Grow(1).MinWidth(0)
		if !whole {
			text.MaxLines(messageLines)
		}
		ui.Text(c, a.messageAge(now, r.Time)).FontSize(caption).TextColor(muted).Shrink(0)
	})
}
