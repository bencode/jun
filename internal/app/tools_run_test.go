package app

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

func shown(a *app, id terminal.ID) bool {
	return slices.Contains(slices.Collect(maps.Values(a.bindings)), id)
}

func TestRunStartsATerminalOfTheCallersProject(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	caller, other, _ := testTerminals(t, a)
	dir := a.state.CurrentProject().CWD
	count := len(a.terminals.All())

	id, err := a.toolRun(tool.Request{Op: tool.OpRun, Command: "printf 'ran %s\\n' it", Dir: dir}, shellOf(t, caller))
	if err != nil {
		t.Fatal(err)
	}
	s := a.terminals.Get(id)
	if s.ProjectName != caller.ProjectName || shown(a, id) {
		t.Fatalf("run: project %s, shown %v", s.ProjectName, shown(a, id))
	}
	eventually(t, "the command to run", func() bool { return strings.Contains(s.Terminal.Text(), "\nran it") })

	// --open shows it, for the terminal the user is with only.
	a.lastTerminal = other.ID
	open := tool.Request{Op: tool.OpRun, Command: "true", Dir: dir, Open: true}
	if _, err := a.toolRun(open, shellOf(t, caller)); err == nil || err.Error() != "the user is not with you" {
		t.Fatalf("run --open for another terminal: %v", err)
	}
	if len(a.terminals.All()) != count+1 {
		t.Fatal("a refused run started a terminal")
	}
	focus(t, a, tt, caller)
	id, err = a.toolRun(open, shellOf(t, caller))
	if err != nil || !a.terminalOnScreen(id) {
		t.Fatalf("run --open: %v, on screen %v", err, a.terminalOnScreen(id))
	}
}

func TestLoopRunsInATerminalOfItsOwnUntilCancelled(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	caller, _, _ := testTerminals(t, a)
	t.Setenv("SHELL", "/bin/sh") // the test app's shell ignores -l -c
	dir := a.state.CurrentProject().CWD

	// Runs repeat, each after the one before ends.
	repeating, err := a.terminals.CreateRunning("Jun", dir, loopProgram(time.Second, "echo tick"), nil)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "two runs", func() bool { return strings.Count(repeating.Terminal.Text(), "\ntick") >= 2 })
	if !strings.Contains(repeating.Terminal.Text(), "── exit 0 · next in 1s") {
		t.Fatalf("no exit status shown:\n%s", repeating.Terminal.Text())
	}

	if _, err := a.toolLoop(tool.Request{Op: tool.OpLoop, Command: "echo hi", Dir: dir, Every: 30 * time.Second}, nil); err == nil {
		t.Fatal("a loop every 30s was taken")
	}
	id, err := a.toolLoop(tool.Request{Op: tool.OpLoop, Command: "echo tock", Dir: dir, Every: time.Minute}, shellOf(t, caller))
	if err != nil {
		t.Fatal(err)
	}
	s := a.terminals.Get(id)
	eventually(t, "the first run", func() bool { return strings.Contains(s.Terminal.Text(), "\ntock") })
	want := []tool.Loop{{ID: uint64(id), Every: "1m", Command: "echo tock", Project: "Jun", Dir: dir}}
	if got := a.toolLoops(); !slices.Equal(got, want) || shown(a, id) {
		t.Fatalf("loops %+v, shown %v", got, shown(a, id))
	}

	if err := a.toolCancelLoop(id); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the loop to end", s.Exited)
	a.reapExitedTerminals(0)
	if a.terminals.Get(id) != nil || len(a.toolLoops()) != 0 {
		t.Fatal("a cancelled loop's terminal stays")
	}
	if err := a.toolCancelLoop(id); err == nil || err.Error() != fmt.Sprintf("no loop %d", id) {
		t.Fatalf("cancel again: %v", err)
	}
}
