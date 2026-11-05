package app

import (
	"cmp"
	"maps"
	"slices"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

const (
	// idleSettle is how long idle must hold to be told: agents go idle for a
	// moment between steps, Codex for a second or two.
	idleSettle = 3 * time.Second
	// eventBuffer is how far a watcher may fall behind.
	eventBuffer = 64
)

// watchedAgents is what watchers know of the agents.
type watchedAgents struct {
	told      map[terminal.ID]tool.Event // each agent's last event, To its status
	idleSince map[terminal.ID]time.Time  // agents gone idle, not told yet
}

// settle compares the agents, as events with their To set, with what
// watchers were told, and returns what they know next and what to tell them.
func settle(w watchedAgents, current map[terminal.ID]tool.Event, now time.Time) (watchedAgents, []tool.Event) {
	next := watchedAgents{told: map[terminal.ID]tool.Event{}, idleSince: map[terminal.ID]time.Time{}}
	var events []tool.Event
	for id, event := range current {
		was, known := w.told[id]
		if known && was.To == event.To {
			next.told[id] = event
			continue
		}
		if known && event.To == string(agent.Idle) {
			since := cmp.Or(w.idleSince[id], now)
			if now.Sub(since) < idleSettle {
				next.told[id], next.idleSince[id] = was, since
				continue
			}
		}
		event.From = was.To
		next.told[id] = event
		events = append(events, event)
	}
	for id, was := range w.told {
		if _, ok := current[id]; !ok {
			was.From, was.To = was.To, ""
			events = append(events, was)
		}
	}
	slices.SortFunc(events, func(x, y tool.Event) int { return cmp.Compare(x.ID, y.ID) })
	return next, events
}

func (a *app) agentEvents() map[terminal.ID]tool.Event {
	events := map[terminal.ID]tool.Event{}
	for id, info := range a.agents {
		if s := a.terminals.Get(id); s != nil {
			events[id] = tool.Event{ID: uint64(id), Project: s.ProjectName, Agent: string(info.Kind),
				Title: a.agentTitle(s), To: string(info.Status)}
		}
	}
	return events
}

// publishAgents tells the watchers what changed, on the UI thread. A watcher
// that fell behind is dropped, for Jun never to wait for one.
func (a *app) publishAgents(now time.Time) {
	var events []tool.Event
	a.watched, events = settle(a.watched, a.agentEvents(), now)
	for queue := range a.watchers {
		if !offer(queue, events) {
			close(queue)
			delete(a.watchers, queue)
		}
	}
}

func offer(queue chan<- tool.Event, events []tool.Event) bool {
	for _, event := range events {
		select {
		case queue <- event:
		default:
			return false
		}
	}
	return true
}

// addWatcher starts a queue from what watchers were last told, on the UI
// thread, for no change to fall between the two.
func (a *app) addWatcher() chan tool.Event {
	told := slices.SortedFunc(maps.Values(a.watched.told), func(x, y tool.Event) int { return cmp.Compare(x.ID, y.ID) })
	queue := make(chan tool.Event, len(told)+eventBuffer)
	for _, event := range told {
		event.From = ""
		queue <- event
	}
	a.watchers[queue] = true
	return queue
}

func (a *app) subscribeAgents(stop <-chan struct{}) (<-chan tool.Event, func()) {
	var queue chan tool.Event
	if !a.onUI(stop, func() { queue = a.addWatcher() }) {
		return nil, func() {} // Jun is closing: the server's closing ends the stream
	}
	return queue, func() { a.window.Update(func() { delete(a.watchers, queue) }) }
}
