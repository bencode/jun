package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// peekHeader is as tall as a region's tab bar: the drawer's terminal is
// then as tall as a full height column's.
const peekHeader = 40

// The drawer takes at most peekShare of the work area, for the rest to
// show dimmed, or peekMinWidth where that is more and the area holds it.
const (
	peekShare    = 0.62
	peekMinWidth = 560
)

// peekWidth keeps the width of the terminal's tab, its grid then staying,
// up to peekShare of area, or peekMinWidth where that is more; a terminal
// never shown takes the most.
func (a *app) peekWidth(id terminal.ID, area ui.Rect) float32 {
	most := max(area.W*peekShare, min(peekMinWidth, area.W-96))
	if size, ok := a.viewSizes[id]; ok {
		return min(size.W, most)
	}
	return most
}

// peekOverlay shows the peeked agent in a drawer at the right of area, the
// work area beside Global, which stays bright; a click on the dimmed rest
// of area closes it.
func (a *app) peekOverlay(c *ui.Context, area ui.Rect) {
	session := a.terminals.Get(a.peek.id)
	if a.peek.id == 0 || session == nil {
		return
	}
	w := a.peekWidth(session.ID, area)
	dim, edge := float32(0.30), a.theme.Accent
	if theme.IsDark(a.theme) {
		dim = 0.55
	}
	if a.isWaiting(session.ID) {
		edge = a.statusColor(agent.Waiting)
	}
	ui.Overlay(c, func() {
		ui.Box(c.Key("agent-peek-backdrop")).Absolute().Left(area.X).Top(area.Y).Width(area.W).Height(area.H).
			Background(ui.RGB(0, 0, 0).Alpha(dim)).OnClick(a.closePeek)
		// One drawer across agents: closing it gives the focus back to
		// where it was before the first. Only its colors change: a size
		// on the way would resize the agent.
		ui.Column(c.Key("agent-peek")).Absolute().Left(area.X + area.W - w).Top(area.Y).Width(w).Height(area.H).
			Background(a.theme.Surface).Transition(ui.ElementTransition{Duration: 120 * time.Millisecond,
			Colors: true, Enter: &ui.Motion{X: 24}}).Children(func() {
			a.peekHeading(c, session)
			term := ghostty.View(c.Key(session.ID), session.Terminal).Width(w).Height(area.H - peekHeader).Shrink(0).
				Label(fmt.Sprintf("%s %d", a.text("Peek terminal", "Peek 终端"), session.ID))
			if a.peek.focus && a.peek.shown {
				term.Focus()
				a.peek.focus = false
			}
			if !a.peek.shown {
				a.peek.shown = true
				c.Invalidate()
			}
			if term.FocusWithin() && a.lastTerminal != session.ID {
				a.lastTerminal = session.ID
				c.Invalidate()
			}
			// An edge over the terminal's padding, not beside it: the
			// terminal keeps the drawer's width.
			ui.Box(c).Absolute().Left(0).Top(0).Bottom(0).Width(2).Background(edge)
		})
	})
}

// peekHeading names the agent, says where Escape goes, and offers the
// drawer's ways on, each with its key.
func (a *app) peekHeading(c *ui.Context, s *terminal.Session) {
	info := a.agents[s.ID]
	status := a.statusName(info.Status)
	if info.Status == agent.Waiting {
		waiting := slices.DeleteFunc(a.agentOrder(), func(id terminal.ID) bool { return !a.isWaiting(id) })
		status = fmt.Sprintf("%s %d/%d", status, slices.Index(waiting, s.ID)+1, len(waiting))
	}
	next := a.text("Next", "下一个")
	if nextWaiting(a.agentOrder(), a.isWaiting, s.ID) == 0 {
		next = a.text("Close", "关闭")
	}
	escape, escapeColor := a.text("Esc → agent", "Esc 发给 Agent"), a.theme.TextMuted
	if a.peek.escaped {
		escape = fmt.Sprintf(a.text("Esc went to the agent · %s closes", "Esc 已发给 Agent · %s 关闭"), closeTabKeys)
		escapeColor = a.statusColor(agent.Waiting)
	}
	heading := ui.Row(c).FillWidth().Height(peekHeader).Shrink(0).Padding(0, 8, 0, 14).Gap(8).AlignItems(ui.Center).
		Background(a.theme.SurfaceHover)
	if info.Status == agent.Waiting {
		heading.Background(a.statusColor(agent.Waiting).Alpha(0.14).Over(a.theme.SurfaceHover))
	}
	heading.Children(func() {
		ui.Box(c).Width(8).Height(8).Radius(4).Shrink(0).Background(a.statusColor(info.Status))
		a.agentNumber(c, s)
		ui.Text(c, a.agentTitle(s)).Bold().MinWidth(0).SingleLine().Ellipsis("…")
		ui.Text(c, s.ProjectName+" · "+agentName(info.Kind)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).
			MinWidth(0).SingleLine().Ellipsis("…")
		ui.Text(c, status).FontSize(theme.Caption).Shrink(0).TextColor(a.statusColor(info.Status))
		ui.Box(c).Height(22).Padding(0, 8).Radius(11).Shrink(0).Border(1, escapeColor).Center().Children(func() {
			ui.Text(c, escape).FontSize(theme.Caption).TextColor(escapeColor)
		})
		ui.Spacer(c)
		a.peekButton(c, fmt.Sprintf("%s %s", next, peekKeys), a.peekNext)
		a.peekButton(c, a.text("Open Tab ⌘↩", "打开标签页 ⌘↩"), a.openPeekedTab)
		closeButton := iconButton(c, closeIcon, fmt.Sprintf(a.text("Close Peek (%s)", "关闭 Peek (%s)"), closeTabKeys), 28, 14).
			Radius(6).OnClick(a.closePeek)
		if a.peek.escaped {
			closeButton.Background(a.theme.Accent.Alpha(0.3))
		}
		ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(1).Background(a.theme.Border)
	})
}

func (a *app) peekButton(c *ui.Context, label string, run func()) {
	ui.ButtonBase(c).Height(26).Padding(0, 8).Radius(6).Shrink(0).Border(1, a.theme.Border).Label(label).
		Children(func() { ui.Text(c, label).FontSize(theme.Caption) }).OnClick(run)
}
