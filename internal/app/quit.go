package app

import (
	"cmp"
	"fmt"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	"github.com/egoist/mygo"
)

// holdClose reports whether closing the window, and so quitting, waits for
// the user to confirm ending the programs still running in terminals,
// asking them.
func (a *app) holdClose(e *mygo.CloseEvent) bool {
	if a.quitConfirmed {
		return false
	}
	busy, err := a.busyTerminals()
	if err == nil && len(busy) == 0 {
		return false
	}
	if err != nil {
		log.Printf("Jun: check running programs: %v", err)
	}
	e.PreventDefault()
	a.confirm(a.text("Quit Jun?", "退出 Jun？"), a.quitDetail(busy, err), a.text("Quit", "退出"), func() {
		a.quitConfirmed = true
		a.window.Close()
	})
	a.window.Show()
	a.window.Invalidate()
	return true
}

// busyTerminals lists the live terminals running something other than
// their shell, with that command.
func (a *app) busyTerminals() (map[terminal.ID]string, error) {
	shells, ttys, unknown := map[terminal.ID]int{}, map[terminal.ID]string{}, map[terminal.ID]int{}
	for _, s := range a.terminals.All() {
		select {
		case <-s.Terminal.Done():
			continue
		default:
		}
		pid, ok := s.ShellPID()
		if !ok {
			continue
		}
		shells[s.ID] = pid
		if tty, ok := a.ttys[s.ID]; ok {
			ttys[s.ID] = tty
		} else {
			unknown[s.ID] = pid
		}
	}
	byPID, err := agent.TTYs(slices.Collect(maps.Values(unknown)))
	if err != nil {
		return nil, err
	}
	for id, pid := range unknown {
		if tty, ok := byPID[pid]; ok {
			ttys[id] = tty
		}
	}
	byTTY := map[string]int{}
	for id, tty := range ttys {
		byTTY[tty] = shells[id]
	}
	commands, err := agent.Busy(byTTY)
	if err != nil {
		return nil, err
	}
	busy := map[terminal.ID]string{}
	for id, tty := range ttys {
		if command, ok := commands[tty]; ok {
			busy[id] = command
		}
	}
	return busy, nil
}

// quitDetail names the programs quitting ends, a line for each terminal,
// by project and then as opened.
func (a *app) quitDetail(busy map[terminal.ID]string, err error) string {
	if err != nil {
		return fmt.Sprintf(a.text("Jun could not check for programs running in terminals (%v). Quitting ends them all.",
			"无法确认终端里是否有程序在运行（%v）。退出会结束所有终端里的程序。"), err)
	}
	sessions := slices.DeleteFunc(a.terminals.All(), func(s *terminal.Session) bool {
		_, ok := busy[s.ID]
		return !ok
	})
	project := func(name string) int {
		return slices.IndexFunc(a.state.Projects, func(p *layout.Project) bool { return p.Name == name })
	}
	slices.SortStableFunc(sessions, func(x, y *terminal.Session) int {
		return cmp.Compare(project(x.ProjectName), project(y.ProjectName))
	})
	lines := []string{fmt.Sprintf(a.text("Programs still run in %d terminals; quitting ends them:", "这 %d 个终端里还有程序在运行，退出会结束它们："), len(sessions))}
	for _, s := range sessions {
		lines = append(lines, fmt.Sprintf("%s — %s · %s", busy[s.ID], s.ProjectName, a.sessionTitle(s)))
	}
	return strings.Join(lines, "\n")
}
