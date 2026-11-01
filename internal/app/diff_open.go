package app

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/terminal"
	"github.com/egoist/mygo/ui"
)

// openDiff shows a source's changes where new tabs go, in the current
// perspective: in its Diff view there, if it has one.
func (a *app) openDiff(project, source string) {
	target := a.tabsTarget()
	if target == nil {
		return
	}
	if _, err := a.state.OpenDiff(target.ID, project, layout.DiffRef{Source: source}); err != nil {
		a.report(err)
		return
	}
	a.layoutChanged()
}

// reviewChanges opens the palette on the current project's branches, to
// open one's changes.
func (a *app) reviewChanges() {
	a.openPalette(a.reviewPrefix())
	a.palette.caretEnd = true
}

func (a *app) reviewPrefix() string { return a.text("Review Changes: ", "查看改动：") }

// reviewItems are the palette's items opening the changes of the current
// project's branches: those checked out first, each said where, and with
// the agent working there.
func (a *app) reviewItems() []paletteItem {
	project := a.state.CurrentProject()
	if project == nil {
		return nil
	}
	r := a.repoFor(project.CWD)
	agents := map[string]string{}
	for _, s := range a.agentSessions() {
		if source, ok := sourceAt(r.sources, agentDir(s)); ok && s.ProjectName == project.Name && agents[source.Key()] == "" {
			info := a.agents[s.ID]
			agents[source.Key()] = agentName(info.Kind) + " · " + a.statusName(info.Status)
		}
	}
	sources := slices.Clone(r.sources)
	unchecked := func(s repo.Source) int {
		if s.Worktree == "" {
			return 1
		}
		return 0
	}
	slices.SortStableFunc(sources, func(x, y repo.Source) int { return cmp.Compare(unchecked(x), unchecked(y)) })
	items := make([]paletteItem, 0, len(sources))
	for _, s := range sources {
		where := ""
		if s.Worktree == r.top {
			where = a.text("project directory", "项目目录")
		} else if s.Worktree != "" {
			where = "worktree"
		}
		detail := strings.Join(slices.DeleteFunc([]string{where, agents[s.Key()]}, func(d string) bool { return d == "" }), " · ")
		items = append(items, paletteItem{title: a.reviewPrefix() + sourceName(s), detail: detail,
			run: func() { a.openDiff(project.Name, s.Key()) }})
	}
	return items
}

// agentMenu offers the changes of the worktree a current project's agent
// works in.
func (a *app) agentMenu(m *ui.Menu, s *terminal.Session) {
	project := a.state.CurrentProject()
	if project == nil || s.ProjectName != project.Name {
		return
	}
	if !m.Item(a.text("Review Changes", "查看改动")).Chosen() {
		return
	}
	dir := agentDir(s)
	source, ok := sourceAt(a.repoFor(project.CWD).sources, dir)
	if !ok {
		a.report(fmt.Errorf("%s is in no worktree of %s", dir, project.Name))
		return
	}
	a.openDiff(project.Name, source.Key())
}

// agentDir is where the agent's shell is, as it reported, else where it
// started; with symlinks followed, as the repository's paths are.
func agentDir(s *terminal.Session) string {
	dir := s.CWD
	if s.Terminal != nil {
		dir = cmp.Or(s.Terminal.Dir(), dir)
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	return dir // as it is, when it cannot be resolved
}

// diffSourceName names a source by its key: its branch, or a detached
// worktree's directory, by its name.
func diffSourceName(key string) string {
	if filepath.IsAbs(key) {
		return filepath.Base(key)
	}
	return key
}

// sourceAt is the source checked out where dir is: of the worktrees
// holding it, the innermost, as worktrees can sit in the main one.
func sourceAt(sources []repo.Source, dir string) (repo.Source, bool) {
	best, found := repo.Source{}, false
	for _, s := range sources {
		inside := s.Worktree != "" && (dir == s.Worktree || strings.HasPrefix(dir, s.Worktree+string(filepath.Separator)))
		if inside && len(s.Worktree) > len(best.Worktree) {
			best, found = s, true
		}
	}
	return best, found
}
