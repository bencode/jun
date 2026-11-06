package app

import (
	"cmp"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
	ghostty "github.com/egoist/mygo/plugins/terminal"
)

// startTools serves the agent tools at a socket beside the configuration,
// telling the terminals started from now on where it is. Without a
// configuration path, already reported, there is none.
//
// A Jun started in another's terminal inherits that one's variables: they
// go first, for its terminals to reach no Jun rather than another one when
// it serves no tools.
func (a *app) startTools(configPath string, stop <-chan struct{}) {
	for _, name := range []string{tool.EnvAPI, tool.EnvBin, tool.EnvTerminal} {
		if err := os.Unsetenv(name); err != nil {
			log.Printf("Jun: unset %s: %v", name, err)
		}
	}
	if configPath == "" {
		return
	}
	socket := filepath.Join(filepath.Dir(configPath), "jun.sock")
	bin, err := os.Executable()
	if err == nil {
		a.tools, err = tool.Start(socket, func(req tool.Request, callers []int) tool.Response {
			return a.handleTool(stop, req, callers)
		}, func() (<-chan tool.Event, func()) { return a.subscribeAgents(stop) })
	}
	if err != nil {
		a.report(fmt.Errorf("%s: %w", a.text("Agent tools unavailable", "Agent 工具不可用"), err))
		return
	}
	a.terminals.SetEnv(func(id terminal.ID) []string { return tool.Env(socket, bin, uint64(id)) })
}

func (a *app) stopTools() {
	if a.tools == nil {
		return
	}
	if err := a.tools.Close(); err != nil {
		log.Printf("Jun: stop agent tools: %v", err)
	}
}

// handleTool runs on the connection's goroutine: it reads Jun's state on
// the UI thread, and a terminal's text off it, as the agent poller does.
func (a *app) handleTool(stop <-chan struct{}, req tool.Request, callers []int) tool.Response {
	onUI := func(fn func()) bool { return a.onUI(stop, fn) }
	closing := tool.Response{Error: "Jun is closing"}
	layout := slices.Contains([]tool.Op{tool.OpOpen, tool.OpClose, tool.OpBack, tool.OpForward}, req.Op)
	refused := func() bool { return layout && a.toolGuard(a.toolCaller(callers), req.Target) != nil }
	if (layout || req.Op == tool.OpMessage) && !a.awaitRepo(stop, onUI, req.Target, refused) {
		return closing
	}
	if req.Op == tool.OpMessage && req.Target != nil && req.Target.Kind == tool.TargetFile && !a.await(stop, onUI, a.reposLoading) {
		return closing
	}
	if answer, ok := a.uiAnswers(req, callers)[req.Op]; ok {
		var resp tool.Response
		if !onUI(func() { resp = answer() }) {
			return closing
		}
		return resp
	}
	if req.Op == tool.OpSend {
		return a.handleSend(stop, onUI, req)
	}
	if req.Op != tool.OpScreen {
		return tool.Response{Error: fmt.Sprintf("unknown op %q", req.Op)}
	}
	if req.Lines < 1 || req.Lines > tool.MaxLines {
		return tool.Response{Error: fmt.Sprintf("lines must be 1 to %d", tool.MaxLines)}
	}
	var source *ghostty.Terminal
	var err error
	if !onUI(func() { source, err = a.toolScreen(terminal.ID(req.Terminal)) }) {
		return closing
	}
	if err != nil {
		return tool.Response{Error: err.Error()}
	}
	return tool.Response{Screen: tool.Tail(source.Text(), req.Lines)}
}

// onUI runs fn on the UI thread and waits for it, unless Jun closes first.
func (a *app) onUI(stop <-chan struct{}, fn func()) bool {
	done := make(chan struct{})
	a.window.Update(func() {
		fn()
		close(done)
	})
	select {
	case <-done:
		return true
	case <-stop:
		return false
	}
}

// uiAnswers answer the ops that one call on the UI thread does.
func (a *app) uiAnswers(req tool.Request, callers []int) map[tool.Op]func() tool.Response {
	outcome := func(err error) tool.Response {
		if err != nil {
			return tool.Response{Error: err.Error()}
		}
		return tool.Response{}
	}
	started := func(id terminal.ID, err error) tool.Response {
		if err != nil {
			return outcome(err)
		}
		return tool.Response{Terminal: uint64(id)}
	}
	layout := func() tool.Response { return a.toolLayout(req, callers) }
	return map[tool.Op]func() tool.Response{
		tool.OpTerminals: func() tool.Response { return tool.Response{Terminals: a.toolTerminals(callers)} },
		tool.OpOpen:      layout,
		tool.OpClose:     layout,
		tool.OpBack:      layout,
		tool.OpForward:   layout,
		tool.OpMessage: func() tool.Response {
			id, err := a.toolMessage(req, callers)
			if err != nil {
				return outcome(err)
			}
			return tool.Response{Created: id}
		},
		tool.OpMessages:      func() tool.Response { return tool.Response{Messages: a.toolMessages()} },
		tool.OpUpdateMessage: func() tool.Response { return outcome(a.toolUpdateMessage(req, callers)) },
		tool.OpRun:           func() tool.Response { return started(a.toolRun(req, callers)) },
		tool.OpLoop:          func() tool.Response { return started(a.toolLoop(req, callers)) },
		tool.OpLoops:         func() tool.Response { return tool.Response{Loops: a.toolLoops()} },
		tool.OpCancelLoop:    func() tool.Response { return outcome(a.toolCancelLoop(terminal.ID(req.Terminal))) },
	}
}

// handleSend sends, then waits up to a few seconds for the agent to work,
// as its status, polled every second, tells.
func (a *app) handleSend(stop <-chan struct{}, onUI func(func()) bool, req tool.Request) tool.Response {
	var err error
	if !onUI(func() { err = a.toolSend(req) }) {
		return tool.Response{Error: "Jun is closing"}
	}
	if err != nil {
		return tool.Response{Error: err.Error()}
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		working := false
		if !onUI(func() { working = a.agentWorking(terminal.ID(req.Terminal)) }) {
			return tool.Response{Error: "Jun is closing"}
		}
		if working {
			return tool.Response{}
		}
		select {
		case <-time.After(200 * time.Millisecond):
		case <-stop:
			return tool.Response{Error: "Jun is closing"}
		}
	}
	return tool.Response{Message: "sent, not started yet"}
}

// awaitRepo waits, up to a few seconds, for the branches of a Diff
// target's repository to be read, as Jun reads them in the background
// once a project's repository is first asked for. Jun reads them again
// once when the terminal is in none of the worktrees read: agents make
// worktrees as they go. Not for a call refused anyway. False: Jun closed.
func (a *app) awaitRepo(stop <-chan struct{}, onUI func(func()) bool, target *tool.Target, refused func() bool) bool {
	if target == nil || target.Kind != tool.TargetDiff {
		return true
	}
	reread := false
	loading := func() bool {
		if refused() {
			return false // refused as it is, at once
		}
		s, err := a.toolTerminal(target.ID)
		if err != nil {
			return false
		}
		_, err = a.toolDiffSource(s)
		if errors.Is(err, errNoWorktree) && !reread {
			reread = true
			dir := a.project(s.ProjectName).CWD
			a.refreshRepo(a.repoFor(dir), dir)
			return true
		}
		return errors.Is(err, errRepoLoading)
	}
	return a.await(stop, onUI, loading) // the change itself then says the repository is still loading
}

// reposLoading tells whether a project's worktrees are still being read: a
// file's project may be told only by them. It starts the reads not begun.
func (a *app) reposLoading() bool {
	return slices.ContainsFunc(a.state.Projects, func(p *layout.Project) bool { return a.repoFor(p.CWD).loading })
}

// await polls, up to a few seconds, until loading says no. False: Jun closed.
func (a *app) await(stop <-chan struct{}, onUI func(func()) bool, loading func() bool) bool {
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		still := false
		if !onUI(func() { still = loading() }) {
			return false
		}
		if !still {
			return true
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-stop:
			return false
		}
	}
	return true
}

// toolTerminals lists the running terminals by id.
func (a *app) toolTerminals(callers []int) []tool.Terminal {
	caller := a.toolCaller(callers)
	var result []tool.Terminal
	for _, s := range a.terminals.All() {
		if s.Exited() {
			continue
		}
		t := tool.Terminal{ID: uint64(s.ID), Project: s.ProjectName, Dir: cmp.Or(s.Terminal.Dir(), s.CWD),
			Title: a.sessionTitle(s), Self: s.ID == caller}
		if info, ok := a.agents[s.ID]; ok {
			t.Title, t.Agent, t.Status = a.agentTitle(s), string(info.Kind), string(info.Status)
		}
		result = append(result, t)
	}
	return result
}

func (a *app) toolScreen(id terminal.ID) (*ghostty.Terminal, error) {
	s := a.terminals.Get(id)
	if s == nil || s.Exited() {
		return nil, fmt.Errorf("no terminal %d", id)
	}
	return s.Terminal, nil
}

// toolCaller is the terminal whose shell the caller descends from, or 0.
func (a *app) toolCaller(callers []int) terminal.ID {
	for _, s := range a.terminals.All() {
		if pid, ok := s.ShellPID(); ok && slices.Contains(callers, pid) {
			return s.ID
		}
	}
	return 0
}
