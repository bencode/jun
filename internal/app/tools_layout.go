package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// toolLayout changes what Jun shows for the caller, on the UI thread. The
// requests are checked first, for their errors to go to the caller rather
// than to a dialog in front of the user.
func (a *app) toolLayout(req tool.Request, callers []int) tool.Response {
	message, err := a.toolChange(req, a.toolCaller(callers))
	if err != nil {
		return tool.Response{Error: err.Error()}
	}
	return tool.Response{Message: message}
}

func (a *app) toolChange(req tool.Request, caller terminal.ID) (string, error) {
	if err := a.toolGuard(caller, req.Target); err != nil {
		return "", err
	}
	if req.Op == tool.OpBack || req.Op == tool.OpForward {
		move, way := layout.History.GoBack, "back"
		if req.Op == tool.OpForward {
			move, way = layout.History.GoForward, "forward"
		}
		if !a.step(move) {
			return "", fmt.Errorf("nothing to go %s to", way)
		}
		return "", nil
	}
	if req.Target == nil {
		return "", errors.New("no target")
	}
	if req.Op == tool.OpOpen {
		return "", a.toolOpen(req)
	}
	return a.toolClose(*req.Target)
}

// toolGuard lets a caller change what the user sees only while the user
// is with it, and spoke to it last, and Jun waits for no answer of the
// user's.
func (a *app) toolGuard(caller terminal.ID, target *tool.Target) error {
	if caller == 0 {
		return errors.New("caller is not a terminal of this Jun")
	}
	if a.dialogOpen() {
		return errors.New("Jun is waiting for the user")
	}
	if caller != a.peek.id && caller != a.lastTerminal {
		return errors.New("the user is not with you")
	}
	if s := a.terminals.Get(caller); s != nil && s.Sent() {
		return errors.New("your last input came from another agent, not the user")
	}
	if target != nil && target.Kind == tool.TargetTerminal && terminal.ID(target.ID) == caller {
		return errors.New("that is your own terminal")
	}
	return nil
}

func (a *app) dialogOpen() bool {
	return a.palette.open || a.projectForm.open || a.perspectiveForm.open || a.projectRename.open ||
		a.shortcutsPanel.open || a.confirmation.open || a.errorOpen ||
		a.persistence != nil && a.persistence.errorOpen
}

func (a *app) toolOpen(req tool.Request) error {
	t := *req.Target
	if t.Kind == tool.TargetProject {
		i := slices.IndexFunc(a.state.Projects, func(p *layout.Project) bool { return strings.EqualFold(p.Name, t.Name) })
		if i < 0 {
			return fmt.Errorf("no project %s", t.Name)
		}
		a.selectProject(a.state.Projects[i].Name)
		return nil
	}
	if t.Kind == tool.TargetPerspective {
		return a.toolPerspective(t.Name)
	}
	if t.Kind != tool.TargetTerminal && t.Kind != tool.TargetDiff {
		return fmt.Errorf("cannot open %s", t.Kind)
	}
	s, err := a.toolTerminal(t.ID)
	if err != nil {
		return err
	}
	if t.Kind == tool.TargetDiff {
		return a.toolOpenDiff(s, req.In)
	}
	if !req.Peek {
		a.revealSession(s.ID)
		return nil
	}
	if _, ok := a.agents[s.ID]; !ok {
		return fmt.Errorf("terminal %d runs no agent", s.ID)
	}
	a.peekAgent(s.ID)
	return nil
}

// toolOpenDiff opens the changes of the branch the terminal is on, as its
// agent card's Review Changes does, in the perspective named in, of the
// terminal's project, when given; focused, which closes any peek.
func (a *app) toolOpenDiff(s *terminal.Session, in string) error {
	source, err := a.toolDiffSource(s)
	if err != nil {
		return err
	}
	if in != "" {
		if s.ProjectName != a.state.ActiveProject {
			a.selectProject(s.ProjectName)
		}
		if err := a.toolPerspective(in); err != nil {
			return err
		}
	}
	target := a.tabsTarget()
	if target == nil {
		return errors.New("no region to open the Diff in")
	}
	a.openDiff(s.ProjectName, source.Key())
	a.focusRegion(target.ID)
	return nil
}

func (a *app) toolClose(t tool.Target) (string, error) {
	if t.Kind == tool.TargetPeek {
		if a.peek.id == 0 {
			return "", errors.New("no peek")
		}
		a.closePeek()
		return "", nil
	}
	if t.Kind == tool.TargetPerspective {
		return a.toolClosePerspective(t.Name)
	}
	if t.Kind != tool.TargetTerminal && t.Kind != tool.TargetDiff {
		return "", fmt.Errorf("cannot close %s", t.Kind)
	}
	s, err := a.toolTerminal(t.ID)
	if err != nil {
		return "", err
	}
	return "", a.toolCloseTabs(t.Kind, s)
}

// toolCloseTabs closes, in the current perspective's columns and bottom,
// the terminal's tab, or the Diffs of the branch it is on.
func (a *app) toolCloseTabs(kind tool.TargetKind, s *terminal.Session) error {
	p := a.state.CurrentPerspective()
	if p == nil {
		return errors.New("no perspective")
	}
	shows := func(v *layout.View) bool { return a.bindings[v.ID] == s.ID }
	missing := fmt.Errorf("terminal %d has no tab here", s.ID)
	if kind == tool.TargetDiff {
		source, err := a.toolDiffSource(s)
		if err != nil {
			return err
		}
		shows = func(v *layout.View) bool {
			return v.Kind == layout.ViewDiff && v.ProjectName == s.ProjectName && v.Diff != nil && v.Diff.Source == source.Key()
		}
		missing = fmt.Errorf("no Diff of %s here", diffSourceName(source.Key()))
	}
	closed := false
	for _, r := range append(slices.Clone(p.Columns), p.Bottom) {
		for _, v := range slices.Clone(r.Views) {
			if shows(v) {
				a.closeView(r.ID, v.ID)
				closed = true
			}
		}
	}
	if !closed {
		return missing
	}
	return nil
}

// toolClosePerspective deletes a perspective of the current project as
// Delete Perspective… does: the user confirms closing terminals in it.
func (a *app) toolClosePerspective(name string) (string, error) {
	project := a.state.CurrentProject()
	if project == nil {
		return "", errors.New("no project")
	}
	i := slices.IndexFunc(project.Perspectives, func(p *layout.Perspective) bool { return strings.EqualFold(p.Name, name) })
	if i < 0 {
		return "", fmt.Errorf("no perspective %s in %s", name, project.Name)
	}
	if len(project.Perspectives) == 1 {
		return "", errors.New("a project keeps at least one perspective")
	}
	a.confirmDeletePerspective(project.Perspectives[i])
	if a.confirmation.open {
		return "waiting for the user to confirm", nil
	}
	return "", nil
}

func (a *app) toolTerminal(id uint64) (*terminal.Session, error) {
	s := a.terminals.Get(terminal.ID(id))
	if s == nil || s.Exited() {
		return nil, fmt.Errorf("no terminal %d", id)
	}
	return s, nil
}

var (
	// errRepoLoading tells that the branches of a terminal's repository
	// are still being read.
	errRepoLoading = errors.New("the repository is still loading")
	// errNoWorktree tells that the terminal is in none of the worktrees
	// read, which an agent may have made since.
	errNoWorktree = errors.New("in no worktree")
)

// toolDiffSource is the branch, or detached worktree, the terminal's shell
// is in.
func (a *app) toolDiffSource(s *terminal.Session) (repo.Source, error) {
	project := a.project(s.ProjectName)
	if project == nil {
		return repo.Source{}, fmt.Errorf("no project %s", s.ProjectName)
	}
	r := a.repoFor(project.CWD)
	if r.loading {
		return repo.Source{}, errRepoLoading
	}
	if r.err != nil {
		return repo.Source{}, r.err
	}
	dir := agentDir(s)
	source, ok := sourceAt(r.sources, dir)
	if !ok {
		return repo.Source{}, fmt.Errorf("%s is %w of %s", dir, errNoWorktree, project.Name)
	}
	return source, nil
}

// toolPerspective selects the current project's perspective of the name,
// whatever its case, making it as New Perspective… does when there is
// none.
func (a *app) toolPerspective(name string) error {
	project := a.state.CurrentProject()
	if project == nil {
		return errors.New("no project")
	}
	if i := slices.IndexFunc(project.Perspectives, func(p *layout.Perspective) bool { return strings.EqualFold(p.Name, name) }); i >= 0 {
		a.selectPerspective(project.Perspectives[i].ID)
		return nil
	}
	if _, err := a.state.AddPerspective(name); err != nil {
		return err
	}
	a.layoutChanged()
	a.focusCurrentPerspective()
	return nil
}
