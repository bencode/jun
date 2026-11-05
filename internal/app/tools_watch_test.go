package app

import (
	"testing"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

func TestSettleTellsWhatHolds(t *testing.T) {
	const id terminal.ID = 3
	agents := func(status agent.Status) map[terminal.ID]tool.Event {
		return map[terminal.ID]tool.Event{id: {ID: uint64(id), Agent: "codex", To: string(status)}}
	}
	start := time.Unix(1000, 0)
	var w watchedAgents
	step := func(current map[terminal.ID]tool.Event, after time.Duration, want ...[2]string) {
		t.Helper()
		var events []tool.Event
		w, events = settle(w, current, start.Add(after))
		if len(events) != len(want) {
			t.Fatalf("at %v: %+v, want %v", after, events, want)
		}
		for i, event := range events {
			if event.From != want[i][0] || event.To != want[i][1] {
				t.Fatalf("at %v: %+v, want %v", after, event, want[i])
			}
		}
	}
	step(agents(agent.Working), 0, [2]string{"", "working"})
	step(agents(agent.Working), time.Second)
	// Idle for a moment, between steps, is not told.
	step(agents(agent.Idle), 2*time.Second)
	step(agents(agent.Working), 3*time.Second)
	// Idle that holds is, counted from when it began.
	step(agents(agent.Idle), 4*time.Second)
	step(agents(agent.Idle), 6*time.Second)
	step(agents(agent.Idle), 7*time.Second, [2]string{"working", "idle"})
	step(agents(agent.Idle), 8*time.Second)
	// Waiting is told at once, and a new idle agent has nothing to settle.
	step(agents(agent.Waiting), 9*time.Second, [2]string{"idle", "waiting"})
	step(nil, 10*time.Second, [2]string{"waiting", ""})
	step(agents(agent.Idle), 11*time.Second, [2]string{"", "idle"})
}

func TestWatchersStartFromWhatWasToldAndFallBehindAlone(t *testing.T) {
	a, _ := testApp(t)
	s, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.agents[s.ID] = agentInfo{agent.Claude, agent.Working}
	a.publishAgents(time.Now())

	events := a.addWatcher()
	if first := <-events; first.ID != uint64(s.ID) || first.From != "" || first.To != "working" {
		t.Fatalf("snapshot: %+v", first)
	}

	a.agents[s.ID] = agentInfo{agent.Claude, agent.Waiting}
	a.publishAgents(time.Now())
	if next := <-events; next.From != "working" || next.To != "waiting" {
		t.Fatalf("change: %+v", next)
	}

	// A watcher that never reads is dropped: Jun does not wait for it.
	for i := range eventBuffer + 2 {
		status := []agent.Status{agent.Working, agent.Waiting}[i%2]
		a.agents[s.ID] = agentInfo{agent.Claude, status}
		a.publishAgents(time.Now())
	}
	if len(a.watchers) != 0 {
		t.Fatalf("watchers: %d", len(a.watchers))
	}
}
