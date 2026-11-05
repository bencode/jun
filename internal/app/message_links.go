package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/tool"
)

// messageLink is what a kind of resource means to a message.
type messageLink struct {
	// project is the resource's project, or why there is no such resource.
	project func(a *app, t tool.Target, sender *layout.Project) (*layout.Project, error)
	label   func(a *app, m message) string
	live    func(a *app, m message) bool // whether it still leads somewhere
	show    func(a *app, m message, reveal bool) error
}

var messageLinks = map[tool.TargetKind]messageLink{
	tool.TargetTerminal: {terminalProject, terminalLabel, terminalLive, showTerminal},
	tool.TargetDiff:     {diffProject, diffLabel, terminalLive, showDiffLink},
	tool.TargetFile:     {fileLinkProject, fileLabel, projectLive, showFileLink},
	tool.TargetProject:  {namedProject, projectLabel, projectLive, showProject},
}

func projectLive(a *app, m message) bool { return slices.Contains(a.state.Projects, m.Project) }

func terminalLive(a *app, m message) bool {
	_, err := a.toolTerminal(m.Link.ID)
	return err == nil
}

func terminalProject(a *app, t tool.Target, _ *layout.Project) (*layout.Project, error) {
	s, err := a.toolTerminal(t.ID)
	if err != nil {
		return nil, err
	}
	return a.project(s.ProjectName), nil
}

func terminalLabel(_ *app, m message) string { return fmt.Sprintf("#%d %s", m.Link.ID, m.Project.Name) }

func showTerminal(a *app, m message, reveal bool) error {
	id := terminal.ID(m.Link.ID)
	if _, isAgent := a.agents[id]; isAgent {
		a.clickAgent(id, reveal)
		return nil
	}
	a.revealSession(id)
	return nil
}

func diffProject(a *app, t tool.Target, _ *layout.Project) (*layout.Project, error) {
	s, err := a.toolTerminal(t.ID)
	if err != nil {
		return nil, err
	}
	if _, err := a.toolDiffSource(s); err != nil {
		return nil, err
	}
	return a.project(s.ProjectName), nil
}

func diffLabel(a *app, m message) string { return terminalLabel(a, m) + " · Diff" }

// showDiffLink goes to the tab of the Diff, wherever it is, else opens one
// in the terminal's project.
func showDiffLink(a *app, m message, _ bool) error {
	s, err := a.toolTerminal(m.Link.ID)
	if err != nil {
		return err
	}
	source, err := a.toolDiffSource(s)
	if err != nil {
		return err
	}
	if view := a.state.FindDiff(s.ProjectName, source.Key()); view != 0 {
		return a.revealTab(view)
	}
	if !a.enterProject(s.ProjectName) {
		return nil
	}
	return a.toolOpenDiff(s, "")
}

func fileLinkProject(a *app, t tool.Target, sender *layout.Project) (*layout.Project, error) {
	if err := a.checkFile(t.Name); err != nil {
		return nil, err
	}
	return a.fileProject(t.Name, sender), nil
}

// fileProject is the project with the deepest directory or worktree holding
// the path, else the sender's.
func (a *app) fileProject(path string, sender *layout.Project) *layout.Project {
	dir, project, depth := filepath.Dir(path), sender, 0
	for _, p := range a.state.Projects {
		sources := append(slices.Clone(a.repoFor(p.CWD).sources), repo.Source{Worktree: p.CWD})
		if s, ok := sourceAt(sources, dir); ok && len(s.Worktree) > depth {
			project, depth = p, len(s.Worktree)
		}
	}
	return project
}

func fileLabel(_ *app, m message) string { return m.Project.Name + " · " + filepath.Base(m.Link.Name) }

// showFileLink goes to the tab of the file, wherever it is, and reads it
// again, else opens one in the message's project.
func showFileLink(a *app, m message, _ bool) error {
	if err := a.checkFile(m.Link.Name); err != nil {
		return err
	}
	project := m.Project.Name
	n := a.fileAt(project, m.Link.Name)
	if view := a.state.FindFile(project, layout.FileRef{Source: n.source, Path: n.path}); view != 0 {
		if st := a.fileViews[view]; st != nil {
			st.read = false
		}
		return a.revealTab(view)
	}
	if !a.enterProject(project) {
		return nil
	}
	return a.showFile(project, m.Link.Name)
}

func namedProject(a *app, t tool.Target, _ *layout.Project) (*layout.Project, error) {
	i := slices.IndexFunc(a.state.Projects, func(p *layout.Project) bool { return strings.EqualFold(p.Name, t.Name) })
	if i < 0 {
		return nil, errors.New("no project " + t.Name)
	}
	return a.state.Projects[i], nil
}

func projectLabel(_ *app, m message) string { return m.Project.Name }

func showProject(a *app, m message, _ bool) error {
	a.selectProject(m.Project.Name)
	return nil
}
