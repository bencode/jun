package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/roles"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// agentTimer is when Jun next signals the agent in a terminal: every
// interval, or once for none.
type agentTimer struct {
	every time.Duration
	next  time.Time
	text  string
}

// toolUpdateTerminal names the terminal, or drops its name for none, and
// sets or drops its timer. It checks all first, to change nothing when it
// refuses any. A name is no other running terminal's, whatever its case:
// agents find a terminal by it. An agent naming its own terminal gives the
// role of the name its session, or takes it out of its role for none.
func (a *app) toolUpdateTerminal(req tool.Request, callers []int) error {
	s, err := a.toolTerminal(req.Terminal)
	if err != nil {
		return err
	}
	var own roles.Session
	var hasOwn bool
	if req.Name != nil {
		if err := a.checkTerminalName(s, *req.Name); err != nil {
			return err
		}
		if own, hasOwn, err = a.ownSession(s, req, callers); err != nil {
			return err
		}
	}
	timer := req.Every != 0 || !req.At.IsZero()
	if timer {
		if err := a.checkTimer(s, req); err != nil {
			return err
		}
	} else if req.Text != "" {
		return errors.New("a timer's text goes with an interval or a time")
	}
	if req.Name != nil {
		name, book := *req.Name, a.roles
		if hasOwn {
			book, name = takeRole(book, own, name)
		}
		if err := a.setRoles(a.renameRoles(book, s.ID, a.names[s.ID], name, hasOwn)); err != nil {
			return fmt.Errorf("role not kept: %w", err)
		}
		if name == "" {
			delete(a.names, s.ID)
		} else {
			a.names[s.ID] = name
		}
	}
	if req.NoTimer {
		delete(a.timers, s.ID)
	}
	if timer {
		next := req.At
		if next.IsZero() {
			next = time.Now().Add(req.Every)
		}
		a.timers[s.ID] = agentTimer{every: req.Every, next: next, text: req.Text}
	}
	return nil
}

func (a *app) checkTerminalName(s *terminal.Session, name string) error {
	if err := tool.CheckName(name); err != nil {
		return err
	}
	if name == "" { // drops it
		return nil
	}
	for _, other := range a.terminals.All() {
		if other.ID != s.ID && !other.Exited() && strings.EqualFold(a.names[other.ID], name) {
			return fmt.Errorf("name %q is terminal %d's", name, other.ID)
		}
	}
	return nil
}

func (a *app) checkTimer(s *terminal.Session, req tool.Request) error {
	if req.NoTimer {
		return errors.New("a timer cannot be set and dropped at once")
	}
	if req.Every != 0 {
		if err := tool.CheckEvery(req.Every); err != nil {
			return err
		}
	}
	if err := tool.CheckTimerText(req.Text); err != nil {
		return err
	}
	if _, ok := a.agents[s.ID]; !ok {
		return fmt.Errorf("terminal %d runs no agent", s.ID)
	}
	return nil
}

// fireTimers signals each agent whose timer is due once it is idle, as
// watchers are told, with nothing in its input: a signal waits, and comes
// once however many it missed. A timer then goes, or is due an interval on.
// One whose terminal runs no agent now waits for one. It follows
// publishAgents, for watchers to have been told of the agents polled.
func (a *app) fireTimers(now time.Time) {
	for id, t := range a.timers {
		s := a.terminals.Get(id)
		if s == nil || s.Exited() || now.Before(t.next) || a.watched.told[id].To != string(agent.Idle) ||
			len(a.heldSends[id]) > 0 || a.hasDraft(s) {
			continue
		}
		s.Send(timerSignal(t))
		if t.every == 0 {
			delete(a.timers, id)
			continue
		}
		t.next = now.Add(t.every)
		a.timers[id] = t
	}
}

// timerSignal is what the timer submits: a tag telling it from the user's
// words, then its text.
func timerSignal(t agentTimer) string {
	return strings.TrimSpace(fmt.Sprintf("[jun-timer %s] %s", timerLabel(t), t.text))
}

// timerLabel is the timer's interval, or for one that fires once, its time.
func timerLabel(t agentTimer) string {
	if t.every == 0 {
		return t.next.Format("15:04")
	}
	return everyLabel(t.every)
}

func (a *app) toolTimer(id terminal.ID) *tool.Timer {
	t, ok := a.timers[id]
	if !ok {
		return nil
	}
	timer := &tool.Timer{Next: t.next, Text: t.text}
	if t.every != 0 {
		timer.Every = everyLabel(t.every)
	}
	return timer
}
