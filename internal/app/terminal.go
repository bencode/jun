package app

import (
	"fmt"
	"log"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/terminal"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

func (a *app) sessionFor(view layout.ViewID) *terminal.Session {
	return a.terminals.Get(a.bindings[view])
}

func (a *app) hiddenTerminals(projectName string) []*terminal.Session {
	return slices.DeleteFunc(a.terminals.ForProject(projectName), func(session *terminal.Session) bool {
		if session.Exited() {
			return true
		}
		for _, id := range a.bindings {
			if id == session.ID {
				return true
			}
		}
		return false
	})
}

func (a *app) reopenTerminal(target layout.RegionID, id terminal.ID) {
	project := a.state.CurrentProject()
	if project == nil {
		return
	}
	session := a.terminals.Get(id)
	if session == nil || !slices.Contains(a.hiddenTerminals(project.Name), session) {
		a.report(fmt.Errorf("terminal %d is not available to reopen", id))
		return
	}
	view, err := a.state.OpenView(target, layout.ViewTerminal, session.ProjectName)
	if err != nil {
		a.report(err)
		return
	}
	a.bindings[view] = id
	a.layoutChanged()
	a.focusRegion(target)
}

func (a *app) terminalView(c *ui.Context, v *layout.View) {
	focus := a.state.Region(a.pendingFocus)
	requested := focus != nil && focus.Active == v.ID
	if session := a.sessionFor(v.ID); session != nil {
		element := ghostty.View(c.Key(v.ID), session.Terminal).Fill().
			Label(fmt.Sprintf("%s %d", a.text("Terminal content", "终端内容"), session.ID))
		if requested {
			element.Focus()
			a.pendingFocus = 0
		}
		if element.FocusWithin() {
			a.focusedTerminal = v.ID
			if a.lastTerminal != session.ID {
				a.lastTerminal = session.ID
				c.Invalidate() // an Agents view built earlier this frame marks the old one
			}
		}
		return
	}
	ui.Column(c.Key(v.ID)).Fill().Padding(28).Gap(12).Children(func() {
		err := a.terminalErrors[v.ID]
		if err == nil {
			ui.Text(c, a.text("Starting terminal…", "正在启动终端…"))
			return
		}
		ui.Text(c, a.text("Terminal could not start", "终端无法启动")).Bold()
		ui.Text(c, err.Error())
		retry := ui.Button(c, a.text("Retry", "重试")).OnClick(func() {
			if err := a.startTerminal(v); err != nil {
				log.Printf("retry terminal %d: %v", v.ID, err)
			}
			if r := a.targetRegion(); r != nil {
				a.focusRegion(r.ID)
			}
		})
		if requested {
			retry.Focus()
			a.pendingFocus = 0
		}
	})
}
