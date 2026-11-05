package app

import (
	"cmp"
	"fmt"
	"time"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// The row's measures: messageRow tells from them how wide a text is.
const (
	messageLines           = 4 // of a long text, shown until it is opened
	messagePadding float32 = 8
	messageBorder  float32 = 2
	messageIcon    float32 = 14
	messageGap     float32 = 6
)

// messagesView lists what agents told the user, the latest first.
func (a *app) messagesView(c *ui.Context) {
	c.After(time.Minute) // for the ages to follow
	area := ui.Column(c.Key("messages")).Fill()
	a.messagesWidth = area.LayoutBounds().W
	if a.messagesWidth == 0 {
		c.AnimationFrame() // the rows need it to tell which texts fold
	}
	area.Children(func() {
		if len(a.messages) == 0 {
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Padding(12).Children(func() {
				ui.Text(c, a.text("Agents tell you here, with jun tool message.", "agent 用 jun tool message 发到这里。")).
					TextColor(a.theme.TextMuted)
			})
			return
		}
		a.messageList(c)
	})
}

// messageList keeps the keyboard on the list: its rows take no focus, so
// ↩, → and ← always act on the row chosen, which a click chooses too.
func (a *app) messageList(c *ui.Context) {
	at := func(row int) message { return a.messages[len(a.messages)-1-row] }
	st := &a.messagesList
	if st.Selected == nil {
		a.messageSelected = -1
		st.Selected = &a.messageSelected
	}
	st.Key = func(row int) any { return at(row).ID }
	first, _ := st.Visible()
	st.KeepOffset = first == 0 // the latest shows as it comes; scrolled away, the rows stay
	st.Label = func(row int) string { return at(row).name() }
	now := time.Now()
	clicked := false
	list := ui.List(c, st, len(a.messages), func(row int) {
		if a.messageRow(c, at(row), now, row == a.messageSelected) {
			a.messageSelected, clicked = row, true
		}
	}).Gap(8).Dividers(1, a.theme.Border).Grow(1).FillWidth()
	if clicked {
		list.Focus()
		c.Invalidate()
	}
	chosen := a.messageSelected
	if chosen < 0 || chosen >= len(a.messages) {
		return
	}
	m := at(chosen)
	if list.Submitted() && a.messageLive(m) {
		a.openMessage(m, false)
		c.Invalidate()
	}
	if list.Shortcut(0, ui.KeyRight) {
		a.setMessageWhole(m.ID, true)
		c.Invalidate()
	}
	if list.Shortcut(0, ui.KeyLeft) {
		a.setMessageWhole(m.ID, false)
		c.Invalidate()
	}
}

// messageRow reports a click on the row or on its fold control.
func (a *app) messageRow(c *ui.Context, m message, now time.Time, chosen bool) (clicked bool) {
	width := a.messagesWidth - 2*(messagePadding+messageBorder)
	if m.Title == "" {
		width -= messageIcon + messageGap
	}
	folds := messageFolds(c, m.Text, width)
	border := ui.Transparent // always there, for a peek to move no row
	if m.Link != nil && m.Link.Kind == tool.TargetTerminal && terminal.ID(m.Link.ID) == a.peek.id {
		border = a.theme.Accent
	}
	row := ui.Column(c).FillWidth().Gap(4).Padding(6, messagePadding).Radius(6).Border(messageBorder, border).Children(func() {
		ui.Row(c).FillWidth().Gap(messageGap).AlignItems(ui.Start).Children(func() {
			a.messageIcon(c, m.Kind, chosen)
			if m.Title != "" {
				ui.Text(c, m.Title).Bold().SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
			} else {
				messageText(c, m, folds).Grow(1).MinWidth(0)
			}
		})
		if m.Title != "" && m.Text != "" {
			messageText(c, m, folds)
		}
		clicked = a.messageFoot(c, m, now, folds, chosen)
	})
	live := a.messageLive(m)
	if m.Link != nil && !live {
		row.Tooltip(a.text("What it links to is gone", "它链接的内容已不在"))
	}
	if live && row.Clicked() {
		a.openMessage(m, row.ClickModifiers()&ui.Super != 0)
		return true
	}
	return clicked
}

// messageFoot closes the row with its tag, fold control and age, and
// reports a click on the control.
func (a *app) messageFoot(c *ui.Context, m message, now time.Time, folds, chosen bool) (clicked bool) {
	muted := a.theme.TextMuted
	if chosen {
		muted = a.theme.AccentText
	}
	ui.Row(c).FillWidth().Gap(messageGap).AlignItems(ui.Center).Children(func() {
		if m.Link != nil {
			ui.Text(c, messageLinks[m.Link.Kind].label(a, m)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…").
				MinWidth(0).Padding(1, 6).Radius(4).Background(a.theme.Surface)
		}
		if folds {
			clicked = a.foldControl(c, m, muted)
		}
		ui.Spacer(c)
		ui.Text(c, a.messageAge(now, m.Time)).FontSize(theme.Caption).TextColor(muted).Shrink(0)
	})
	return clicked
}

// name is what the message goes by, for the list and assistive technology.
func (m message) name() string { return cmp.Or(m.Title, m.Text) }

// messageFolds tells whether the text runs past messageLines at the width.
func messageFolds(c *ui.Context, text string, width float32) bool {
	if text == "" || width <= 0 {
		return false
	}
	_, whole := c.MeasureText(width, ui.Span{Text: text})
	_, line := c.MeasureText(width, ui.Span{Text: "x"})
	return whole > (messageLines+0.5)*line
}

func messageText(c *ui.Context, m message, folds bool) ui.Element {
	text := ui.Text(c, m.Text).LineHeight(1.4)
	if folds && !m.Whole {
		return text.MaxLines(messageLines)
	}
	return text
}

// foldControl shows or folds a long text, and reports its click, which is
// not the row's.
func (a *app) foldControl(c *ui.Context, m message, color ui.Color) bool {
	label := a.text("Show more", "展开")
	if m.Whole {
		label = a.text("Show less", "收起")
	}
	control := ui.Box(c).Shrink(0).Padding(1, 4).Radius(4).Children(func() {
		ui.Text(c, label).FontSize(theme.Caption).TextColor(color)
	})
	if !control.Clicked() {
		return false
	}
	a.setMessageWhole(m.ID, !m.Whole)
	return true
}

func (a *app) messageIcon(c *ui.Context, kind tool.MessageKind, chosen bool) {
	looks := map[tool.MessageKind]struct {
		icon  *ui.SVG
		color ui.Color
	}{
		tool.MessageAsk:  {askIcon, a.theme.Accent},
		tool.MessageWarn: {warnIcon, a.theme.Warning},
		tool.MessageDone: {doneIcon, a.theme.Success},
		tool.MessageInfo: {infoIcon, a.theme.TextMuted},
	}
	color := looks[kind].color
	if chosen {
		color = a.theme.AccentText // the row's own color would hide the ask icon
	}
	ui.Box(c).Padding(2, 0, 0, 0).Shrink(0).Children(func() { // down to the first line's middle
		ui.Icon(c, looks[kind].icon).Size(messageIcon, messageIcon).TextColor(color)
	})
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
