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
// ↩, →, ← and ⌫ always act on the row chosen, which a click chooses too.
// A message goes once the list is built, whose rows it would move.
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
	clicked, deleted := false, uint64(0)
	list := ui.List(c, st, len(a.messages), func(row int) {
		m := at(row)
		click, remove := a.messageRow(c, m, now, row == a.messageSelected)
		if click {
			a.messageSelected, clicked = row, true
		}
		if remove {
			deleted = m.ID
		}
	}).Gap(8).Dividers(1, a.theme.Border).Grow(1).FillWidth()
	if clicked {
		list.Focus()
		c.Invalidate()
	}
	if deleted != 0 {
		a.deleteMessage(deleted)
		c.Invalidate()
		return
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
	if list.Shortcut(0, ui.KeyBackspace) || list.Shortcut(0, ui.KeyDelete) {
		a.deleteMessage(m.ID)
		c.Invalidate()
	}
}

// messageRow reports a click on the row or on its fold control, and Delete
// chosen in its menu.
func (a *app) messageRow(c *ui.Context, m message, now time.Time, chosen bool) (clicked, deleted bool) {
	width, size := a.messagesWidth-2*(messagePadding+messageBorder), a.messageSize()
	textWidth := width
	if m.Title == "" {
		number, _ := c.MeasureText(0, ui.Span{Text: m.number(), Size: theme.Caption})
		textWidth -= a.messageIconSize() + number + 2*messageGap
	}
	folds := messageFolds(c, m.Text, textWidth, size) || a.repliesFold(c, m, width, now)
	border := ui.Transparent // always there, for a peek to move no row
	if m.Link != nil && m.Link.Kind == tool.TargetTerminal && terminal.ID(m.Link.ID) == a.peek.id {
		border = a.theme.Accent
	}
	row := ui.Column(c).FillWidth().Gap(4).Padding(6, messagePadding).Radius(6).Border(messageBorder, border).ContextMenu(func(menu *ui.Menu) {
		deleted = menu.Item(a.text("Delete", "删除")).Chosen()
	})
	if m.Task != 0 && m.Task == a.selectedTask {
		row.Background(a.taskTint())
	}
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(messageGap).AlignItems(ui.Start).Children(func() {
			a.messageIcon(c, m.Kind, chosen)
			a.messageNumber(c, m, chosen)
			if m.Title != "" {
				ui.Text(c, m.Title).FontSize(size).Bold().SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
			} else {
				messageText(c, m, folds, size).Grow(1).MinWidth(0)
			}
		})
		if m.Title != "" && m.Text != "" {
			messageText(c, m, folds, size)
		}
		a.messageReplies(c, m, now, chosen)
		clicked = a.messageFoot(c, m, now, folds, chosen)
	})
	live := a.messageLive(m)
	if m.Link != nil && !live {
		row.Tooltip(a.text("What it links to is gone", "它链接的内容已不在"))
	}
	if live && row.Clicked() {
		a.openMessage(m, row.ClickModifiers()&ui.Super != 0)
		return true, deleted
	}
	return clicked, deleted
}

// messageFoot closes the row with its tag, fold control and age, and
// reports a click on the control.
func (a *app) messageFoot(c *ui.Context, m message, now time.Time, folds, chosen bool) (clicked bool) {
	muted := a.messageMuted(chosen)
	ui.Row(c).FillWidth().Gap(messageGap).AlignItems(ui.Center).Children(func() {
		if m.Link != nil {
			ui.Text(c, messageLinks[m.Link.Kind].label(a, m)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…").
				MinWidth(0).Padding(1, 6).Radius(4).Background(a.theme.Surface)
		}
		if m.Task != 0 {
			clicked = a.messageTask(c, m.Task) || clicked
		}
		if folds {
			clicked = a.foldControl(c, m, muted) || clicked
		}
		ui.Spacer(c)
		ui.Text(c, a.messageAge(now, m.Time)).FontSize(theme.Caption).TextColor(muted).Shrink(0)
	})
	return clicked
}

// messageTask is the tag of the task the message is about, which shows it
// in Tasks; it reports its click, which is not the row's. A task taken out
// leaves its number.
func (a *app) messageTask(c *ui.Context, id int) bool {
	t, ok := a.goals.Task(id)
	label := taskNumber(id)
	if ok {
		label += " " + t.Title
	}
	tag := ui.Box(c).MinWidth(0).Shrink(1).Padding(1, 6).Radius(4).Background(a.theme.Surface).Children(func() {
		ui.Text(c, label).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine().Ellipsis("…")
	})
	if !ok || !tag.Clicked() {
		return false
	}
	a.revealTask(id)
	return true
}

// name is what the message goes by, for the list and assistive technology.
func (m message) name() string { return cmp.Or(m.Title, m.Text) }

// number is what the user and agents call the message.
func (m message) number() string { return fmt.Sprintf("M%d", m.ID) }

// messageSize is the size of a message's text: the size of Jun's lists,
// as Agents and Tasks have it, not the terminals'.
func (a *app) messageSize() float32 { return a.theme.Rem(1) }

func (a *app) messageIconSize() float32 { return a.messageSize() + 1 }

func (a *app) messageMuted(chosen bool) ui.Color {
	if chosen {
		return a.theme.AccentText
	}
	return a.theme.TextMuted
}

func (a *app) messageNumber(c *ui.Context, m message, chosen bool) {
	ui.Box(c).Padding((a.messageSize()-theme.Caption)/2, 0, 0, 0).Shrink(0).Children(func() { // down to the first line's middle
		ui.Text(c, m.number()).FontSize(theme.Caption).TextColor(a.messageMuted(chosen))
	})
}

// messageFolds tells whether the text runs past messageLines at the width.
func messageFolds(c *ui.Context, text string, width, size float32) bool {
	if text == "" || width <= 0 {
		return false
	}
	_, whole := c.MeasureText(width, ui.Span{Text: text, Size: size})
	_, line := c.MeasureText(width, ui.Span{Text: "x", Size: size})
	return whole > (messageLines+0.5)*line
}

func messageText(c *ui.Context, m message, folds bool, size float32) ui.Element {
	text := ui.Text(c, m.Text).FontSize(size).LineHeight(1.4)
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
		ui.Icon(c, looks[kind].icon).Size(a.messageIconSize(), a.messageIconSize()).TextColor(color)
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
