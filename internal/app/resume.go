package app

import (
	"fmt"
	"os"
	"slices"

	"github.com/bencode/jun/internal/history"
	"github.com/bencode/jun/internal/layout"
)

// resumeSession opens a terminal where new tabs go, in the session's
// directory, and types the agent's command that resumes it: the shell
// stays when the agent exits.
func (a *app) resumeSession(project string, s history.Session) {
	command, err := a.resumeIn(s)
	if err != nil {
		a.report(err)
		return
	}
	target := a.tabsTarget()
	if target == nil {
		return
	}
	id, err := a.state.OpenView(target.ID, layout.ViewTerminal, project)
	if err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
	a.focusRegion(target.ID)
	views := a.state.Region(target.ID).Views
	view := views[slices.IndexFunc(views, func(v *layout.View) bool { return v.ID == id })]
	if err := a.startTerminalIn(view, s.CWD); err != nil {
		a.report(err)
		return
	}
	a.sessionFor(id).Terminal.Send([]byte(command + "\r"))
}

// resumeIn is the command that resumes the session in its directory, as
// the settings say the agent starts there; refused when the directory is
// gone.
func (a *app) resumeIn(s history.Session) (string, error) {
	command, err := history.ResumeCommand(s, a.resumeCommands())
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(s.CWD); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s: %s", a.text("The session's directory is gone", "会话的目录已不存在"), s.CWD)
	}
	return command, nil
}
