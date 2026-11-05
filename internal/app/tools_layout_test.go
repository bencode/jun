package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
	"github.com/egoist/mygo/ui"
)

// shellOf is the process a program in the session's terminal descends
// from, as the server finds it among the caller's ancestors.
func shellOf(t *testing.T, s *terminal.Session) []int {
	t.Helper()
	var pid int
	eventually(t, "the shell's process", func() bool {
		var ok bool
		pid, ok = s.ShellPID()
		return ok
	})
	return []int{pid}
}

// layoutTool calls the tool from the session's terminal, then shows a
// frame, which records the move.
func layoutTool(t *testing.T, a *app, tt *ui.Tester, from *terminal.Session, req tool.Request) tool.Response {
	t.Helper()
	resp := a.toolLayout(req, shellOf(t, from))
	tt.Frame()
	return resp
}

// focus gives the session's terminal the keyboard: the user is with it.
func focus(t *testing.T, a *app, tt *ui.Tester, s *terminal.Session) {
	t.Helper()
	a.revealSession(s.ID)
	tt.Frame()
	if a.lastTerminal != s.ID {
		t.Fatalf("terminal %d did not take the focus", s.ID)
	}
}

func target(kind tool.TargetKind, id terminal.ID, name string) *tool.Target {
	return &tool.Target{Kind: kind, ID: uint64(id), Name: name}
}

// testTerminals are testApp's terminals: Work's first column's and second
// column's, and Review's bottom one.
func testTerminals(t *testing.T, a *app) (first, second, review *terminal.Session) {
	t.Helper()
	work, other := a.state.CurrentProject().Perspectives[0], a.state.CurrentProject().Perspectives[1]
	first, second, review = a.sessionFor(work.Columns[0].Active), a.sessionFor(work.Columns[1].Active), a.sessionFor(other.Bottom.Active)
	if first == nil || second == nil || review == nil {
		t.Fatal("testApp has no terminals where expected")
	}
	return first, second, review
}

func TestLayoutToolsChangeOnlyForTheTerminalTheUserIsWith(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	caller, other, _ := testTerminals(t, a)
	back := tool.Request{Op: tool.OpBack}
	refused := func(req tool.Request, want string) {
		t.Helper()
		if resp := layoutTool(t, a, tt, caller, req); resp.Error != want {
			t.Fatalf("%+v: %q, want %q", req, resp.Error, want)
		}
	}
	if resp := a.toolLayout(back, nil); resp.Error != "caller is not a terminal of this Jun" {
		t.Fatalf("a caller outside Jun: %q", resp.Error)
	}
	a.lastTerminal = other.ID
	refused(back, "the user is not with you")
	a.peek = agentPeek{id: caller.ID}
	refused(back, "nothing to go back to") // past the guard
	a.peek, a.lastTerminal = agentPeek{}, caller.ID
	refused(tool.Request{Op: tool.OpOpen, Target: target(tool.TargetTerminal, caller.ID, "")}, "that is your own terminal")
	a.palette.open = true
	refused(back, "Jun is waiting for the user")
}

func TestLayoutToolsOpenAndGoBack(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	caller, _, review := testTerminals(t, a)
	focus(t, a, tt, caller)
	call := func(req tool.Request) {
		t.Helper()
		if resp := layoutTool(t, a, tt, caller, req); resp.Error != "" {
			t.Fatalf("%+v: %s", req, resp.Error)
		}
	}
	perspective := func() string { return a.state.CurrentPerspective().Name }

	call(tool.Request{Op: tool.OpOpen, Target: target(tool.TargetPerspective, 0, "review")})
	if perspective() != "Review" || len(a.state.CurrentProject().Perspectives) != 2 {
		t.Fatalf("open perspective:review is in %s", perspective())
	}
	call(tool.Request{Op: tool.OpBack})
	if perspective() != "Work" {
		t.Fatalf("back went to %s", perspective())
	}
	call(tool.Request{Op: tool.OpOpen, Target: target(tool.TargetPerspective, 0, "Notes")})
	if perspective() != "Notes" || len(a.state.CurrentProject().Perspectives) != 3 {
		t.Fatalf("open perspective:Notes is in %s", perspective())
	}
	call(tool.Request{Op: tool.OpBack})

	// Another terminal's tab takes the user there, and to it.
	call(tool.Request{Op: tool.OpOpen, Target: target(tool.TargetTerminal, review.ID, "")})
	if perspective() != "Review" || !a.terminalOnScreen(review.ID) || a.lastTerminal != review.ID {
		t.Fatalf("open terminal:%d is in %s, last terminal %d", review.ID, perspective(), a.lastTerminal)
	}
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpBack}); resp.Error != "the user is not with you" {
		t.Fatalf("back once the user is with another terminal: %q", resp.Error)
	}

	focus(t, a, tt, caller)
	peek := tool.Request{Op: tool.OpOpen, Target: target(tool.TargetTerminal, review.ID, ""), Peek: true}
	if resp := layoutTool(t, a, tt, caller, peek); resp.Error != fmt.Sprintf("terminal %d runs no agent", review.ID) {
		t.Fatalf("peek of a shell: %q", resp.Error)
	}
	a.agents[review.ID] = agentInfo{Kind: agent.Claude, Status: agent.Idle}
	if resp := a.toolLayout(peek, shellOf(t, caller)); resp.Error != "" || a.peek.id != review.ID {
		t.Fatalf("peek: %q, peeking %d", resp.Error, a.peek.id)
	}
}

func TestLayoutToolsClose(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	caller, other, _ := testTerminals(t, a)
	focus(t, a, tt, caller)
	closing := func(target *tool.Target) tool.Response {
		t.Helper()
		return layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpClose, Target: target})
	}

	if resp := closing(target(tool.TargetTerminal, other.ID, "")); resp.Error != "" || a.terminalOnScreen(other.ID) || other.Exited() {
		t.Fatalf("close terminal: %q; on screen %v, exited %v", resp.Error, a.terminalOnScreen(other.ID), other.Exited())
	}
	if resp := closing(target(tool.TargetTerminal, other.ID, "")); resp.Error != fmt.Sprintf("terminal %d has no tab here", other.ID) {
		t.Fatalf("close a closed tab: %q", resp.Error)
	}

	if resp := closing(target(tool.TargetPeek, 0, "")); resp.Error != "no peek" {
		t.Fatalf("close no peek: %q", resp.Error)
	}
	a.peek = agentPeek{id: other.ID}
	if resp := closing(target(tool.TargetPeek, 0, "")); resp.Error != "" || a.peek.id != 0 {
		t.Fatalf("close peek: %q", resp.Error)
	}

	// Review's terminal makes the user confirm; Work then is the last.
	if resp := closing(target(tool.TargetPerspective, 0, "review")); resp.Message != "waiting for the user to confirm" || !a.confirmation.open {
		t.Fatalf("close perspective:review: %+v", resp)
	}
	a.confirmation.open = false
	a.confirmation.run()
	if resp := closing(target(tool.TargetPerspective, 0, "Work")); resp.Error != "a project keeps at least one perspective" {
		t.Fatalf("close the last perspective: %q", resp.Error)
	}
}

func TestLayoutToolsOpenAndCloseTheDiffOfATerminalsBranch(t *testing.T) {
	a := testNewApp(t)
	if err := a.state.AddProject("Jun", gitRepo(t)); err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(a.view, 1440, 900)
	tt.Frame()
	caller, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	other, err := a.terminals.Create("Jun", a.state.CurrentProject().CWD, nil)
	if err != nil {
		t.Fatal(err)
	}
	a.lastTerminal = caller.ID
	diff := target(tool.TargetDiff, other.ID, "")

	resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpOpen, Target: diff, In: "Review"})
	p := a.state.CurrentPerspective()
	if v := activeView(p.Columns[0]); resp.Error != "" || p.Name != "Review" || v == nil || v.Kind != layout.ViewDiff || v.Diff.Source != "main" {
		t.Fatalf("open diff: %q, in %s", resp.Error, p.Name)
	}
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpClose, Target: diff}); resp.Error != "" || len(p.Columns[0].Views) != 0 {
		t.Fatalf("close diff: %q", resp.Error)
	}
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpClose, Target: diff}); resp.Error != "no Diff of main here" {
		t.Fatalf("close a closed diff: %q", resp.Error)
	}

	// A worktree an agent made since Jun read the repository.
	worktree := filepath.Join(t.TempDir(), "feature")
	if out, err := exec.Command("git", "-C", a.state.CurrentProject().CWD, "worktree", "add", "-q", "-b", "feature", worktree).CombinedOutput(); err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, out)
	}
	agentTerminal, err := a.terminals.Create("Jun", worktree, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := tool.Request{Op: tool.OpOpen, Target: target(tool.TargetDiff, agentTerminal.ID, "")}
	if resp := a.toolLayout(req, shellOf(t, caller)); !strings.Contains(resp.Error, "in no worktree") {
		t.Fatalf("before reading the repository again: %q", resp.Error)
	}
	onUI := func(fn func()) bool { fn(); return true }
	if !a.awaitRepo(nil, onUI, req.Target, func() bool { return false }) {
		t.Fatal("awaitRepo: Jun closed")
	}
	if resp := a.toolLayout(req, shellOf(t, caller)); resp.Error != "" || activeView(a.tabsTarget()).Diff.Source != "feature" {
		t.Fatalf("open diff of the new worktree: %q", resp.Error)
	}
}

func TestLayoutToolsOpenAndCloseFiles(t *testing.T) {
	a, tt := testApp(t)
	tt.Frame()
	caller, other, _ := testTerminals(t, a)
	path := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := target(tool.TargetFile, 0, path)
	focus(t, a, tt, other)
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpOpen, Target: file}); resp.Error != "the user is not with you" {
		t.Fatalf("from another terminal: %q", resp.Error)
	}
	focus(t, a, tt, caller)
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpOpen, Target: file}); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if v := activeView(a.tabsTarget()); v.File == nil || v.File.Path != "report.txt" {
		t.Fatalf("the file did not open: %+v", v)
	}
	focus(t, a, tt, caller)
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpClose, Target: file}); resp.Error != "" {
		t.Fatal(resp.Error)
	}
	focus(t, a, tt, caller)
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpClose, Target: file}); !strings.Contains(resp.Error, "no tab of") {
		t.Fatalf("closed twice: %q", resp.Error)
	}
	missing := target(tool.TargetFile, 0, path+".gone")
	if resp := layoutTool(t, a, tt, caller, tool.Request{Op: tool.OpOpen, Target: missing}); !strings.Contains(resp.Error, "no file") {
		t.Fatalf("a missing file: %q", resp.Error)
	}
}
