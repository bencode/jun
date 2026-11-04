package app

import (
	"fmt"
	"slices"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// agentPeek is the agent shown in a drawer out of Global's edge: its own
// terminal, borrowed while its tab is off screen, as a terminal shows in
// one view at a time. ⌘ keys stay Jun's there; every other key, Escape
// too, goes to the agent, as in any terminal pane.
type agentPeek struct {
	id    terminal.ID // 0 while closed
	focus bool        // the terminal takes the focus once the drawer shows
	// shown tells that the drawer showed a frame: the overlay keeps the
	// element focused as it came, to give the focus back to, so the
	// terminal takes it only after.
	shown bool
	// escaped tells that Escape went to the agent a moment ago, the hint
	// then saying where the way out is; escapes counts them, for the
	// hint to go only after the last.
	escaped bool
	escapes int
}

// peekHeader is as tall as a region's tab bar: the drawer's terminal is
// then as tall as a full height column's.
const peekHeader = 40

// The drawer takes at most peekShare of the work area, for the rest to
// show dimmed, or peekMinWidth where that is more and the area holds it.
const (
	peekShare    = 0.62
	peekMinWidth = 560
	escapeHint   = 2 * time.Second
)

var peekKeys = keyCombo{ui.Super, ui.KeyJ}

// peekAgent peeks the agent, or focuses its tab when that shows already.
func (a *app) peekAgent(id terminal.ID) {
	if a.terminalOnScreen(id) {
		a.closePeek()
		a.revealSession(id)
		return
	}
	if id != a.peek.id {
		a.peek = agentPeek{id: id, shown: a.peek.shown}
	}
	a.peek.focus = true
}

func (a *app) closePeek() { a.peek = agentPeek{} }

// peekNext goes to the next waiting agent, closing once no other waits.
// Closed, it peeks the first waiting agent but the focused one, else the
// current agent.
func (a *app) peekNext() {
	order := a.agentOrder()
	if a.peek.id != 0 {
		if next := nextWaiting(order, a.isWaiting, a.peek.id); next != 0 {
			a.peekAgent(next)
		} else {
			a.closePeek()
		}
		return
	}
	var focused terminal.ID
	if s := a.sessionFor(a.focusedTerminal); s != nil {
		focused = s.ID
	}
	if next := nextWaiting(order, a.isWaiting, focused); next != 0 {
		a.peekAgent(next)
	} else if _, ok := a.agents[a.lastTerminal]; ok {
		a.peekAgent(a.lastTerminal)
	}
}

// openPeekedTab closes the drawer for the agent's own tab.
func (a *app) openPeekedTab() {
	id := a.peek.id
	a.closePeek()
	a.revealSession(id)
}

// agentOrder is the agents as the Agents view lists them.
func (a *app) agentOrder() []terminal.ID {
	sessions := a.agentSessions()
	order := make([]terminal.ID, len(sessions))
	for i, s := range sessions {
		order[i] = s.ID
	}
	return order
}

func (a *app) isWaiting(id terminal.ID) bool { return a.agents[id].Status == agent.Waiting }

// peekEscaped shows, a moment, that Escape went to the peeked agent.
func (a *app) peekEscaped(id terminal.ID) {
	if id == 0 || id != a.peek.id {
		return
	}
	a.peek.escaped = true
	a.peek.escapes++
	if a.window == nil {
		return
	}
	a.window.Invalidate()
	escapes := a.peek.escapes
	time.AfterFunc(escapeHint, func() {
		a.window.Update(func() {
			if a.peek.id == id && a.peek.escapes == escapes {
				a.peek.escaped = false
			}
		})
	})
}

// nextWaiting returns the first waiting agent after after in order,
// wrapping around, other than after itself; 0 when there is none.
func nextWaiting(order []terminal.ID, waiting func(terminal.ID) bool, after terminal.ID) terminal.ID {
	start := slices.Index(order, after) + 1
	rotated := slices.Concat(order[start:], order[:start])
	i := slices.IndexFunc(rotated, func(id terminal.ID) bool { return id != after && waiting(id) })
	if i < 0 {
		return 0
	}
	return rotated[i]
}

// terminalOnScreen reports whether the session's tab is the active one of
// a region the current perspective shows.
func (a *app) terminalOnScreen(id terminal.ID) bool {
	p := a.state.CurrentPerspective()
	if p == nil {
		return false
	}
	regions := slices.Clone(p.Columns)
	if p.BottomVisible {
		regions = append(regions, p.Bottom)
	}
	if p.SideVisible {
		regions = append(regions, p.Side)
	}
	return slices.ContainsFunc(regions, func(r *layout.Region) bool {
		bound, ok := a.bindings[r.Active]
		return ok && bound == id
	})
}

// dropStalePeek closes the peek before a frame shows its terminal twice,
// or once the layout asks for the focus, the agent quits or its terminal
// ends.
func (a *app) dropStalePeek() {
	id := a.peek.id
	if id == 0 {
		return
	}
	_, isAgent := a.agents[id]
	session := a.terminals.Get(id)
	if a.pendingFocus != 0 || !isAgent || session == nil || session.Exited() || a.terminalOnScreen(id) {
		a.closePeek()
	}
}

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
