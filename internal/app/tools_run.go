package app

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// loop is a command a terminal runs again and again, every interval
// after a run ends.
type loop struct {
	every   time.Duration
	command string
	dir     string
}

// toolRun starts a terminal in the caller's project and types the command
// into its shell, which stays once the command ends, as resuming an agent
// session does. Shown in a tab only when asked, for the terminal the user
// is with.
func (a *app) toolRun(req tool.Request, callers []int) (terminal.ID, error) {
	if strings.TrimSpace(req.Command) == "" {
		return 0, errors.New("no command")
	}
	caller := a.toolCaller(callers)
	if req.Open {
		if err := a.toolGuard(caller, nil); err != nil {
			return 0, err
		}
	}
	project, err := a.newTerminalProject(caller)
	if err != nil {
		return 0, err
	}
	s, err := a.terminals.Create(project, req.Dir, a.invalidate())
	if err != nil {
		return 0, err
	}
	s.Terminal.Send([]byte(req.Command + "\r"))
	if req.Open {
		a.revealSession(s.ID)
	}
	return s.ID, nil
}

// toolLoop starts a terminal of the caller's project running the command
// every interval, with no tab.
func (a *app) toolLoop(req tool.Request, callers []int) (terminal.ID, error) {
	if strings.TrimSpace(req.Command) == "" {
		return 0, errors.New("no command")
	}
	if req.Every < tool.MinEvery || req.Every%time.Second != 0 {
		return 0, fmt.Errorf("interval must be whole seconds, at least %s", tool.MinEvery)
	}
	project, err := a.newTerminalProject(a.toolCaller(callers))
	if err != nil {
		return 0, err
	}
	s, err := a.terminals.CreateRunning(project, req.Dir, loopProgram(req.Every, req.Command), a.invalidate())
	if err != nil {
		return 0, err
	}
	a.loops[s.ID] = loop{every: req.Every, command: req.Command, dir: req.Dir}
	return s.ID, nil
}

// toolLoops lists the loops whose terminals run, forgetting the others.
func (a *app) toolLoops() []tool.Loop {
	maps.DeleteFunc(a.loops, func(id terminal.ID, _ loop) bool {
		s := a.terminals.Get(id)
		return s == nil || s.Exited()
	})
	var result []tool.Loop
	for _, id := range slices.Sorted(maps.Keys(a.loops)) {
		l := a.loops[id]
		result = append(result, tool.Loop{ID: uint64(id), Every: everyLabel(l.every), Command: l.command,
			Project: a.terminals.Get(id).ProjectName, Dir: l.dir})
	}
	return result
}

// toolCancelLoop ends the loop's process group, as closing its terminal
// would: the terminal then ends as a shell that exits does.
func (a *app) toolCancelLoop(id terminal.ID) error {
	s := a.terminals.Get(id)
	if _, ok := a.loops[id]; !ok || s == nil || s.Exited() {
		delete(a.loops, id)
		return fmt.Errorf("no loop %d", id)
	}
	pid, ok := s.ShellPID()
	if !ok {
		return fmt.Errorf("loop %d has not started", id)
	}
	if err := syscall.Kill(-pid, syscall.SIGHUP); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("end loop %d: %w", id, err)
	}
	delete(a.loops, id)
	return nil
}

// newTerminalProject is the project of the caller's terminal, else the
// current one.
func (a *app) newTerminalProject(caller terminal.ID) (string, error) {
	if s := a.terminals.Get(caller); s != nil {
		return s.ProjectName, nil
	}
	if p := a.state.CurrentProject(); p != nil {
		return p.Name, nil
	}
	return "", errors.New("no project")
}

func (a *app) invalidate() func() {
	if a.window == nil {
		return nil
	}
	return a.window.Invalidate
}

// loopProgram runs the command every interval after a run ends, with the
// user's shell as a script runs it, each run's time and exit status shown.
func loopProgram(every time.Duration, command string) []string {
	const script = `printf '\033]0;loop %s: %s\007' "$2" "$3"
while :; do
  printf '\n── %s · %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$3"
  "${SHELL:-/bin/zsh}" -l -c "$3"
  printf '── exit %d · next in %s\n' "$?" "$2"
  sleep "$1"
done`
	return []string{"/bin/sh", "-c", script, "jun-loop", strconv.Itoa(int(every / time.Second)), everyLabel(every), command}
}

// everyLabel writes an interval as it is typed: 5m, not 5m0s.
func everyLabel(every time.Duration) string {
	s := every.String()
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}
