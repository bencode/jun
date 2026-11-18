package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

func TestLeadsFromRunAndJoinLastWhileTheLeadRuns(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	lead, worker, shell := testTerminals(t, a)
	brain := t.TempDir()
	if err := a.state.AddProject("Brain", brain); err != nil {
		t.Fatal(err)
	}
	a.agents[lead.ID] = agentInfo{agent.Claude, agent.Working}
	listed := func(id terminal.ID) tool.Terminal {
		t.Helper()
		for _, terminal := range a.toolTerminals(nil, 0) {
			if terminal.ID == uint64(id) {
				return terminal
			}
		}
		t.Fatalf("terminal %d not listed", id)
		return tool.Terminal{}
	}

	// An agent's run leads the terminal, in the project asked for, in its
	// directory; a shell's leads none.
	ran, err := a.toolRun(tool.Request{Op: tool.OpRun, Command: "true", Project: "brain"}, shellOf(t, lead))
	if err != nil {
		t.Fatal(err)
	}
	if s := a.terminals.Get(ran); s.ProjectName != "Brain" || s.CWD != brain || listed(ran).Lead != uint64(lead.ID) {
		t.Fatalf("run by an agent: project %s, dir %s, %+v", s.ProjectName, s.CWD, listed(ran))
	}
	plain, err := a.toolRun(tool.Request{Op: tool.OpRun, Command: "true", Dir: brain}, shellOf(t, shell))
	if err != nil || listed(plain).Lead != 0 || a.terminals.Get(plain).ProjectName != "Jun" {
		t.Fatalf("run by a shell: %v, %+v", err, listed(plain))
	}
	if _, err := a.toolRun(tool.Request{Op: tool.OpRun, Command: "true", Project: "Nowhere"}, shellOf(t, lead)); err == nil {
		t.Fatal("run in no project was taken")
	}
	a.agents[ran] = agentInfo{agent.Codex, agent.Waiting}
	if got := a.agentEvents()[ran].Lead; got != uint64(lead.ID) {
		t.Fatalf("watch tells lead %d", got)
	}
	if label := a.agentLabel(a.terminals.Get(ran)); !strings.Contains(label, fmt.Sprintf("#%d", lead.ID)) {
		t.Fatalf("the agent's label names no lead: %q", label)
	}

	join := func(from *terminal.Session, to terminal.ID) error {
		return a.toolJoin(tool.Request{Op: tool.OpJoin, Terminal: uint64(to)}, shellOf(t, from))
	}
	for _, refused := range []struct {
		from *terminal.Session
		to   terminal.ID
		want string
	}{
		{worker, shell.ID, fmt.Sprintf("terminal %d runs no agent", shell.ID)},
		{lead, lead.ID, "a terminal cannot join itself"},
		{lead, ran, fmt.Sprintf("terminal %d is led by you", ran)},
	} {
		if err := join(refused.from, refused.to); err == nil || err.Error() != refused.want {
			t.Fatalf("join %d from %d: %v, want %q", refused.to, refused.from.ID, err, refused.want)
		}
	}
	// A join takes the place of the lead before; leave drops it.
	if err := join(worker, lead.ID); err != nil || listed(worker.ID).Lead != uint64(lead.ID) {
		t.Fatalf("join: %v, %+v", err, listed(worker.ID))
	}
	if err := join(worker, ran); err != nil || listed(worker.ID).Lead != uint64(ran) {
		t.Fatalf("join another: %v, %+v", err, listed(worker.ID))
	}
	if err := a.toolLeave(tool.Request{Op: tool.OpLeave}, shellOf(t, worker)); err != nil || listed(worker.ID).Lead != 0 {
		t.Fatalf("leave: %v, %+v", err, listed(worker.ID))
	}

	// A lead whose terminal ends leads nothing, those it led none either.
	if err := join(worker, ran); err != nil {
		t.Fatal(err)
	}
	lead.Terminal.Send([]byte("exit\n"))
	eventually(t, "the lead's shell to exit", lead.Exited)
	if a.lead(ran) != 0 || a.lead(worker.ID) != ran {
		t.Fatalf("after the lead ended: %d led by %d, %d by %d", ran, a.lead(ran), worker.ID, a.lead(worker.ID))
	}
	a.reapExitedTerminals(0)
	if _, ok := a.leads[ran]; ok {
		t.Fatal("a lead that ended is kept")
	}
}
