package app

import (
	"cmp"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"

	"github.com/bencode/jun/internal/agent"
	"github.com/bencode/jun/internal/layout"
)

func (a *app) startTerminal(v *layout.View) error { return a.startTerminalIn(v, "") }

// startTerminalIn starts the terminal view's shell in dir; when dir is "",
// where its shell was when Jun last quit, else in its project's directory.
func (a *app) startTerminalIn(v *layout.View, dir string) error {
	if v.Kind != layout.ViewTerminal {
		return fmt.Errorf("view %d is not a terminal", v.ID)
	}
	if a.sessionFor(v.ID) != nil {
		return nil
	}
	var invalidate func()
	if a.window != nil {
		invalidate = a.window.Invalidate
	}
	for _, project := range a.state.Projects {
		if project.Name != v.ProjectName {
			continue
		}
		session, err := a.terminals.Create(project.Name, cmp.Or(dir, existingDir(v.Dir), project.CWD), invalidate)
		if err != nil {
			a.terminalErrors[v.ID] = err
			return err
		}
		a.bindings[v.ID] = session.ID
		delete(a.terminalErrors, v.ID)
		return nil
	}
	err := fmt.Errorf("source project not found: %s", v.ProjectName)
	a.terminalErrors[v.ID] = err
	return err
}

func (a *app) restoreTerminals() {
	for _, view := range a.terminalViews() {
		if err := a.startTerminal(view); err != nil {
			log.Printf("restore terminal %d: %v", view.ID, err)
		}
	}
}

// terminalViews are the terminal views of every region, hidden ones and
// other projects' included.
func (a *app) terminalViews() []*layout.View {
	regions := []*layout.Region{a.state.Global}
	for _, project := range a.state.Projects {
		for _, p := range project.Perspectives {
			regions = append(regions, p.Columns...)
			regions = append(regions, p.Bottom)
		}
	}
	var views []*layout.View
	for _, region := range regions {
		views = append(views, slices.DeleteFunc(slices.Clone(region.Views), func(v *layout.View) bool { return v.Kind != layout.ViewTerminal })...)
	}
	return views
}

// recordTerminalDirs notes on each terminal view where its shell is, for
// the next start; a shell Jun cannot read keeps the note it had.
func (a *app) recordTerminalDirs() {
	shells := map[*layout.View]int{}
	for _, v := range a.terminalViews() {
		if s := a.sessionFor(v.ID); s != nil && !s.Exited() {
			if pid, ok := s.ShellPID(); ok {
				shells[v] = pid
			}
		}
	}
	dirs, err := agent.CWDs(slices.Collect(maps.Values(shells)))
	if err != nil {
		log.Printf("Jun: note terminals' directories: %v", err)
		return
	}
	for v, pid := range shells {
		if dir, ok := dirs[pid]; ok && dir != v.Dir {
			v.Dir = dir
			a.layoutChanged()
		}
	}
}

// existingDir is dir while it is a directory, else "".
func existingDir(dir string) string {
	if info, err := os.Stat(dir); dir == "" || err != nil || !info.IsDir() {
		return ""
	}
	return dir
}

func (a *app) closeTerminals() {
	if err := a.terminals.CloseAll(); err != nil {
		log.Printf("close terminals: %v", err)
	}
}
