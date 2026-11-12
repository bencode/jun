package app

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

func eventually(t *testing.T, what string, ready func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); !ready(); {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestToolTerminalsTellTheCallerAndSkipExited(t *testing.T) {
	a, _ := testApp(t)
	a.terminals.SetEnv(func(id terminal.ID) []string { return tool.Env("/tmp/jun.sock", "/bin/jun", uint64(id)) })
	caller, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	caller.Terminal.Send([]byte("printf '\\nAPI=%s TERMINAL=%s\\n' \"$JUN_API\" \"$JUN_TERMINAL\"\n"))
	want := fmt.Sprintf("\nAPI=/tmp/jun.sock TERMINAL=%d", caller.ID)
	eventually(t, "the shell's environment", func() bool { return strings.Contains(caller.Terminal.Text(), want) })
	pid, ok := caller.ShellPID()
	if !ok {
		t.Fatal("the shell did not report its process")
	}

	self := func(callers []int) []uint64 {
		var ids []uint64
		for _, terminal := range a.toolTerminals(callers, 0) {
			if terminal.Self {
				ids = append(ids, terminal.ID)
			}
		}
		return ids
	}
	// A program in the shell is a caller with the shell among its ancestors.
	if got := self([]int{999999, pid, 1}); !slices.Equal(got, []uint64{uint64(caller.ID)}) {
		t.Fatalf("self: %v, want %d", got, caller.ID)
	}
	if got := self(nil); len(got) != 0 {
		t.Fatalf("a caller outside Jun's terminals is no terminal: %v", got)
	}

	caller.Terminal.Send([]byte("exit\n"))
	eventually(t, "the shell to exit", caller.Exited)
	if slices.ContainsFunc(a.toolTerminals(nil, 0), func(t tool.Terminal) bool { return t.ID == uint64(caller.ID) }) {
		t.Fatal("an exited terminal is listed")
	}
	if _, err := a.toolScreen(caller.ID); err == nil {
		t.Fatal("screen of an exited terminal should fail")
	}
	if _, err := a.toolScreen(999); err == nil || err.Error() != "no terminal 999" {
		t.Fatalf("screen of no terminal: %v", err)
	}
}

func TestTerminalsInheritNoOtherJunsTools(t *testing.T) {
	a, _ := testApp(t)
	t.Setenv(tool.EnvAPI, "/other/jun.sock") // as in another Jun's terminal
	a.startTools("", nil)                    // no tools served
	s, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Terminal.Send([]byte("printf '\\nAPI=[%s]\\n' \"$JUN_API\"\n"))
	eventually(t, "the shell's environment", func() bool { return strings.Contains(s.Terminal.Text(), "\nAPI=[") })
	if !strings.Contains(s.Terminal.Text(), "\nAPI=[]") {
		t.Fatalf("the terminal reaches another Jun: %s", s.Terminal.Text())
	}
}
