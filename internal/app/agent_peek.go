package app

import (
	"slices"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
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

// escapeHint is how long the drawer says that Escape went to the agent.
const escapeHint = 2 * time.Second

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
