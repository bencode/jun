package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

func TestTerminalNamesAreUniqueWhileTheTerminalRuns(t *testing.T) {
	a, tt, sessions := agentsApp(t)
	first, second := sessions[0], sessions[1]
	name := func(s *terminal.Session, name string) error {
		return a.toolUpdateTerminal(tool.Request{Op: tool.OpUpdateTerminal, Terminal: uint64(s.ID), Name: &name}, nil)
	}
	listed := func(id terminal.ID) string {
		t.Helper()
		for _, terminal := range a.toolTerminals(nil, 0) {
			if terminal.ID == uint64(id) {
				return terminal.Name
			}
		}
		t.Fatalf("terminal %d not listed", id)
		return ""
	}

	// Named, it is listed, told to watchers and shown in Agents.
	if err := name(first, "达芬奇"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if listed(first.ID) != "达芬奇" || a.agentEvents()[first.ID].Name != "达芬奇" || !tt.HasText("达芬奇") {
		t.Fatalf("named: listed %q, told %q, shown %v", listed(first.ID), a.agentEvents()[first.ID].Name, tt.HasText("达芬奇"))
	}
	if listed(second.ID) != "" {
		t.Fatalf("an unnamed terminal listed as %q", listed(second.ID))
	}

	// Another running terminal's name is refused; its own again is not.
	if err := name(second, "达芬奇"); err == nil || err.Error() != fmt.Sprintf(`name "达芬奇" is terminal %d's`, first.ID) {
		t.Fatalf("a name taken: %v", err)
	}
	if err := name(first, "达芬奇"); err != nil {
		t.Fatalf("its own name again: %v", err)
	}
	// Taken in another case, it is taken.
	if err := name(first, "Leonardo"); err != nil {
		t.Fatal(err)
	}
	if err := name(second, "leonardo"); err == nil {
		t.Fatal("a name taken in another case is not refused")
	}
	// Jun checks a name as the command does, for a request from elsewhere.
	for _, bad := range []string{"42", "a\nb", " a", strings.Repeat("x", tool.MaxName+1)} {
		if err := name(second, bad); err == nil {
			t.Errorf("name %q taken", bad)
		}
	}
	// A new name replaces it, and frees the old; none drops it.
	if err := name(first, "欧拉"); err != nil || listed(first.ID) != "欧拉" {
		t.Fatalf("renamed: %v, %q", err, listed(first.ID))
	}
	if err := name(second, "达芬奇"); err != nil {
		t.Fatalf("the old name, freed: %v", err)
	}
	if err := name(first, ""); err != nil || listed(first.ID) != "" {
		t.Fatalf("dropped: %v, %q", err, listed(first.ID))
	}
	if err := name(&terminal.Session{ID: 999}, "高斯"); err == nil || err.Error() != "no terminal 999" {
		t.Fatalf("no such terminal: %v", err)
	}

	// A terminal that ended gives its name up.
	second.Terminal.Send([]byte("exit\n"))
	eventually(t, "the shell to exit", second.Exited)
	if err := name(first, "达芬奇"); err != nil {
		t.Fatalf("the name of a terminal that ended: %v", err)
	}
	a.reapExitedTerminals(0)
	if _, ok := a.names[second.ID]; ok {
		t.Fatal("the name of a closed terminal is kept")
	}
}

func TestTerminalTimersAreSetOnAgentsOnly(t *testing.T) {
	a, _, sessions := agentsApp(t)
	lead, shell := sessions[0], sessions[1]
	delete(a.agents, shell.ID)
	update := func(req tool.Request) error {
		req.Op = tool.OpUpdateTerminal
		return a.toolUpdateTerminal(req, nil)
	}
	listed := func(id terminal.ID) *tool.Timer {
		t.Helper()
		for _, terminal := range a.toolTerminals(nil, 0) {
			if terminal.ID == uint64(id) {
				return terminal.Timer
			}
		}
		t.Fatalf("terminal %d not listed", id)
		return nil
	}

	before := time.Now()
	if err := update(tool.Request{Terminal: uint64(lead.ID), Every: 30 * time.Minute, Text: "看看新工单"}); err != nil {
		t.Fatal(err)
	}
	if got := listed(lead.ID); got == nil || got.Every != "30m" || got.Text != "看看新工单" || got.Next.Before(before.Add(30*time.Minute)) {
		t.Fatalf("periodic: %+v", got)
	}
	// A new timer replaces it; --no-timer drops it, the name kept.
	at := time.Now().Add(time.Hour)
	name := "达芬奇"
	if err := update(tool.Request{Terminal: uint64(lead.ID), At: at, Name: &name}); err != nil {
		t.Fatal(err)
	}
	if got := listed(lead.ID); got == nil || got.Every != "" || got.Text != "" || !got.Next.Equal(at) {
		t.Fatalf("once: %+v", got)
	}
	if err := update(tool.Request{Terminal: uint64(lead.ID), NoTimer: true}); err != nil || listed(lead.ID) != nil || a.names[lead.ID] != name {
		t.Fatalf("dropped: %v, %+v, %q", err, listed(lead.ID), a.names[lead.ID])
	}

	// Refused, nothing changes: not even the name that goes with it.
	other := "欧拉"
	for req, want := range map[tool.Request]string{
		{Terminal: uint64(shell.ID), Every: time.Minute, Name: &other}:     fmt.Sprintf("terminal %d runs no agent", shell.ID),
		{Terminal: uint64(lead.ID), Every: 30 * time.Second, Name: &other}: "interval must be whole seconds, at least 1m0s",
		{Terminal: uint64(lead.ID), Every: time.Minute, Text: "a\nb"}:      fmt.Sprintf("timer text must be one line of at most %d bytes", tool.MaxTimerText),
		{Terminal: uint64(lead.ID), Text: "ping"}:                          "a timer's text goes with an interval or a time",
		{Terminal: uint64(lead.ID), Every: time.Minute, NoTimer: true}:     "a timer cannot be set and dropped at once",
	} {
		if err := update(req); err == nil || err.Error() != want {
			t.Errorf("%+v: %v, want %q", req, err, want)
		}
	}
	if a.names[shell.ID] != "" || a.names[lead.ID] != name || len(a.timers) != 0 {
		t.Fatalf("changed on refusal: %q, %q, %v", a.names[shell.ID], a.names[lead.ID], a.timers)
	}
}

func TestTimersSignalIdleAgentsOnce(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	_, target, _ := testTerminals(t, a)
	shellOf(t, target)
	shown := 0
	show := func(input string) { // what Claude Code shows: its input box, the user's draft in it or not
		t.Helper()
		shown++
		done := fmt.Sprintf("⏺ Done %d.", shown)
		target.Terminal.Feed([]byte("\x1b[2J\x1b[H" + done + "\r\n──────\r\n❯ " + input + "\r\n──────\r\n"))
		eventually(t, "the input box", func() bool { return strings.Contains(target.Terminal.Text(), done) })
	}
	status := func(s agent.Status, now time.Time) {
		a.agents[target.ID] = agentInfo{Kind: agent.Claude, Status: s}
		a.publishAgents(now)
	}
	t0 := time.Now()
	status(agent.Working, t0.Add(-time.Minute))
	a.timers[target.ID] = agentTimer{every: time.Minute, next: t0, text: "ping"}
	fired := func(now time.Time) bool {
		next := a.timers[target.ID].next
		a.fireTimers(now)
		return !a.timers[target.ID].next.Equal(next)
	}

	show("")
	if fired(t0.Add(-time.Second)) || fired(t0) {
		t.Fatal("fired before due, or while the agent works")
	}
	status(agent.Idle, t0) // idle for a moment, between steps
	if fired(t0) {
		t.Fatal("fired before idle settled")
	}
	status(agent.Idle, t0.Add(idleSettle))
	if !fired(t0.Add(idleSettle)) || !a.timers[target.ID].next.Equal(t0.Add(idleSettle+time.Minute)) {
		t.Fatalf("not fired once idle, or next %v", a.timers[target.ID].next)
	}
	eventually(t, "the signal", func() bool { return strings.Contains(target.Terminal.Text(), "[jun-timer 1m] ping") })

	// Due while the user types: it waits, held nowhere, for the input to empty.
	later := t0.Add(time.Hour)
	show("half a thought")
	if fired(later) || len(a.heldSends[target.ID]) != 0 {
		t.Fatalf("fired over a draft, or held %q", a.heldSends[target.ID])
	}
	show("")
	if !fired(later) {
		t.Fatal("not fired once the input emptied")
	}

	// One that fires once goes once it has.
	a.timers[target.ID] = agentTimer{next: later}
	a.fireTimers(later)
	if _, ok := a.timers[target.ID]; ok {
		t.Fatal("a timer that fires once is kept")
	}
	eventually(t, "the signal", func() bool { return strings.Contains(target.Terminal.Text(), "[jun-timer "+later.Format("15:04")+"]") })

	// A terminal that ended gives its timer up.
	a.timers[target.ID] = agentTimer{every: time.Minute, next: later}
	target.Terminal.Send([]byte("exit\n"))
	eventually(t, "the shell to exit", target.Exited)
	a.reapExitedTerminals(0)
	if _, ok := a.timers[target.ID]; ok {
		t.Fatal("the timer of a closed terminal is kept")
	}
}
