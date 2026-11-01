package app

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var refreshIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7"/></svg>`))

// filesViewState is a Files view's tree: which items are open and chosen,
// and the file it last showed as current; and its search.
type filesViewState struct {
	outline  ui.OutlineState[fileNode]
	selected int
	opened   bool // the first source opened, once
	current  fileNode
	search   fileSearch
}

// filesView lists the sources of the project's repository, each with its
// files, or a search's results in one of them: a click shows a file in a
// preview tab, a double click or Enter in a tab of its own.
func (a *app) filesView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil {
		return
	}
	r := a.repoFor(project.CWD)
	st := a.trees[v.ID]
	if st == nil {
		st = &filesViewState{selected: -1}
		st.outline.List.Selected = &st.selected
		a.trees[v.ID] = st
	}
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		// One row: the search, in a source, and refreshing.
		var s repo.Source
		ui.Row(c).FillWidth().Height(36).Padding(0, 6, 0, 8).Gap(4).AlignItems(ui.Center).Children(func() {
			if len(r.sources) > 0 {
				s = a.searchSource(r, st, project.Name)
				a.fileSearchBar(c, r, st, s)
			} else {
				ui.Spacer(c)
			}
			if iconButton(c, refreshIcon, a.text("Refresh", "刷新"), 24, 14).Clicked() {
				a.refreshRepo(r, project.CWD)
			}
		})
		if r.sources == nil {
			a.filesMessage(c, r, project.CWD)
			return
		}
		if len(r.sources) == 0 {
			return
		}
		if strings.TrimSpace(st.search.query) != "" {
			a.fileSearchResults(c, r, st, s, project.Name)
			return
		}
		if !st.opened {
			st.opened = true
			st.outline.Open.Add(fileNode{source: r.sources[0].Key()})
		}
		a.filesTree(c, r, st, project.Name)
	})
}

// filesMessage says why there is no tree yet: reading, no repository, or
// git's error, with a retry.
func (a *app) filesMessage(c *ui.Context, r *repoState, dir string) {
	ui.Column(c).Padding(12).Gap(8).FillWidth().Children(func() {
		if r.loading {
			ui.Text(c, a.text("Reading…", "正在读取…")).TextColor(a.theme.TextMuted)
			return
		}
		if errors.Is(r.err, repo.ErrNotRepository) {
			ui.Text(c, a.text("Not a Git repository", "不是 Git 仓库")).TextColor(a.theme.TextMuted)
			return
		}
		if r.err != nil {
			ui.Text(c, r.err.Error()).TextColor(a.theme.Danger)
			if ui.Button(c, a.text("Read Again", "重新读取")).Clicked() {
				a.refreshRepo(r, dir)
			}
		}
	})
}

func (a *app) filesTree(c *ui.Context, r *repoState, st *filesViewState, project string) {
	roots := make([]fileNode, len(r.sources))
	for i, s := range r.sources {
		roots[i] = fileNode{source: s.Key()}
	}
	children := func(n fileNode) []fileNode {
		s, ok := r.source(n.source)
		if !ok {
			return nil
		}
		if n.path == "" && !st.outline.Open.Has(n) {
			return []fileNode{} // read when opened
		}
		return a.listingFor(r, s).children[n.path]
	}
	a.fileOutline(c, r, &st.outline, roots, children, project)
	a.followCurrentFile(st, project)
}

// fileOutline shows files and directories: a click shows a file in a
// preview tab, a double click or Enter keeps it in a tab of its own, or
// opens or closes a directory.
func (a *app) fileOutline(c *ui.Context, r *repoState, st *ui.OutlineState[fileNode], roots []fileNode, children func(fileNode) []fileNode, project string) ui.Element {
	outline := ui.Outline(c, st, roots, children, func(n fileNode) {
		a.fileRow(c, r, n, children(n) != nil)
	}).Grow(1).FillWidth()
	chosen := func() (fileNode, bool) {
		i := *st.List.Selected
		if i < 0 || i >= st.Rows() {
			return fileNode{}, false
		}
		n := st.Item(i)
		return n, children(n) == nil
	}
	if outline.Changed() {
		if n, file := chosen(); file {
			a.openFile(project, n, false)
		}
	}
	if outline.Submitted() {
		if n, file := chosen(); file {
			a.openFile(project, n, true)
		} else if st.Open.Has(n) {
			st.Open.Remove(n)
		} else {
			st.Open.Add(n)
		}
	}
	return outline
}

// currentFile is the file the tabs' target shows, if it is the project's.
func (a *app) currentFile(project string) (layout.FileRef, bool) {
	if target := a.tabsTarget(); target != nil {
		if v := activeView(target); v != nil && v.File != nil && v.ProjectName == project {
			return *v.File, true
		}
	}
	return layout.FileRef{}, false
}

// followCurrentFile chooses the row of the file the tabs' target shows, as
// that file changes.
func (a *app) followCurrentFile(st *filesViewState, project string) {
	var current fileNode
	if ref, ok := a.currentFile(project); ok {
		current = fileNode{source: ref.Source, path: ref.Path}
	}
	if current == st.current {
		return
	}
	st.current = current
	for i := range st.outline.Rows() {
		if st.outline.Item(i) == current {
			st.selected = i
		}
	}
}

func (a *app) project(name string) *layout.Project {
	for _, p := range a.state.Projects {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func activeView(r *layout.Region) *layout.View {
	for _, v := range r.Views {
		if v.ID == r.Active {
			return v
		}
	}
	return nil
}

// fileRow is a row of Files: a source, named by its branch and where it is
// read from, or a file or directory.
func (a *app) fileRow(c *ui.Context, r *repoState, n fileNode, dir bool) {
	s, _ := r.source(n.source)
	ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).ContextMenu(func(m *ui.Menu) { a.fileMenu(m, r, s, n) }).Children(func() {
		if n.path != "" {
			ui.Text(c, path.Base(n.path)).MinWidth(0).SingleLine().Ellipsis("…")
			return
		}
		ui.Text(c, sourceName(s)).Bold().MinWidth(0).SingleLine().Ellipsis("…")
		detail := ""
		if s.Worktree != "" && s.Branch != "" {
			detail = filepath.Base(s.Worktree)
		} else if s.Worktree == "" {
			detail = a.text("committed ", "已提交 ") + s.Commit
		} else {
			detail = s.Commit
		}
		// The branch keeps its room; where it is read from gives way first.
		ui.Text(c, detail).FontSize(theme.Caption).TextColor(a.theme.TextMuted).MinWidth(0).Shrink(100).SingleLine().Ellipsis("…")
		if l := r.listings[n.source]; l != nil && l.err != nil {
			ui.Text(c, a.text("unreadable", "读取失败")).FontSize(theme.Caption).TextColor(a.theme.Danger).Tooltip(l.err.Error())
		}
	})
}

// sourceName is a source's branch, or for a detached worktree its
// directory's name.
func sourceName(s repo.Source) string {
	if s.Branch != "" {
		return s.Branch
	}
	return filepath.Base(s.Worktree)
}

// fileMenu copies a row's path, and for a worktree's files shows them in
// Finder or opens them in their default app.
func (a *app) fileMenu(m *ui.Menu, r *repoState, s repo.Source, n fileNode) {
	if n.path != "" && m.Item(a.text("Copy Relative Path", "复制相对路径")).Chosen() {
		mygo.Clipboard.WriteText(n.path)
	}
	if s.Worktree == "" {
		return
	}
	full := filepath.Join(s.Worktree, filepath.FromSlash(n.path))
	if m.Item(a.text("Copy Path", "复制路径")).Chosen() {
		mygo.Clipboard.WriteText(full)
	}
	if m.Item(a.text("Reveal in Finder", "在 Finder 中显示")).Chosen() {
		mygo.Shell.ShowItemInFolder(full)
	}
	if n.path != "" && m.Item(a.text("Open with Default App", "用默认应用打开")).Chosen() {
		if err := mygo.Shell.OpenPath(full); err != nil {
			a.report(err)
		}
	}
}

// openFile shows the file in the region new tabs go to: in its preview
// tab, which the next file replaces, or with pinned in a tab of its own.
// A tab there showing the file already is chosen.
func (a *app) openFile(project string, n fileNode, pinned bool) {
	target := a.tabsTarget()
	if target == nil {
		return
	}
	ref := layout.FileRef{Source: n.source, Path: n.path}
	shown := false
	for _, v := range target.Views {
		shown = shown || v.File != nil && v.ProjectName == project && *v.File == ref
	}
	preview := a.previewTabs[target.ID]
	id, err := a.state.OpenFile(target.ID, project, ref, preview)
	if err != nil {
		a.report(err)
		return
	}
	if pinned && id == preview {
		delete(a.previewTabs, target.ID)
	} else if !pinned && !shown {
		a.previewTabs[target.ID] = id
	}
	a.layoutChanged()
}
