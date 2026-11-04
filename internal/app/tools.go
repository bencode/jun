package app

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

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
		})
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
	onUI := func(fn func()) bool {
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
	closing := tool.Response{Error: "Jun is closing"}
	if req.Op == tool.OpTerminals {
		var terminals []tool.Terminal
		if !onUI(func() { terminals = a.toolTerminals(callers) }) {
			return closing
		}
		return tool.Response{Terminals: terminals}
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
