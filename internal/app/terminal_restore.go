package app

import (
	"cmp"
	"fmt"
	"log"

	"github.com/bencode/jun/internal/layout"
)

func (a *app) startTerminal(v *layout.View) error { return a.startTerminalIn(v, "") }

// startTerminalIn starts the terminal view's shell in dir, or in its
// project's directory when dir is "".
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
		session, err := a.terminals.Create(project.Name, cmp.Or(dir, project.CWD), invalidate)
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
	regions := []*layout.Region{a.state.Global}
	for _, project := range a.state.Projects {
		for _, p := range project.Perspectives {
			regions = append(regions, p.Columns...)
			regions = append(regions, p.Bottom)
		}
	}
	for _, region := range regions {
		for _, view := range region.Views {
			if view.Kind == layout.ViewTerminal {
				if err := a.startTerminal(view); err != nil {
					log.Printf("restore terminal %d: %v", view.ID, err)
				}
			}
		}
	}
}

func (a *app) closeTerminals() {
	if err := a.terminals.CloseAll(); err != nil {
		log.Printf("close terminals: %v", err)
	}
}
