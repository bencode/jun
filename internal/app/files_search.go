package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// fileSearchLimit is the most files a search shows: past it, typing more
// narrows the search better than reading on.
const fileSearchLimit = 500

// branchPrefix begins a search of the branches, by name, rather than of a
// branch's files.
const branchPrefix = "b:"

// fileSearch is a Files view's search: its query, in one source, and the
// files matching it as a tree; or, after branchPrefix, the branches found.
type fileSearch struct {
	query        string
	source       string // the source chosen for it; "" follows the current file
	outline      ui.OutlineState[fileNode]
	selected     int
	made         searchKey // what the results were made for
	results      map[string][]fileNode
	more         bool                      // more files match than the results show
	focusResults bool                      // choose the first file, and focus the results
	pinFirst     bool                      // and keep it in a tab of its own
	branches     ui.OutlineState[fileNode] // the branches found, by day
	branch       int                       // the one chosen
}

type searchKey struct {
	query, source string
	listing       *listing
	reads         int
}

// matchFiles returns the files whose paths hold every word of the query,
// ignoring case, in their order, at most limit of them; more reports that
// others match too.
func matchFiles(files []string, query string, limit int) (matches []string, more bool) {
	words := strings.Fields(strings.ToLower(query))
	for _, f := range files {
		if !holdsWords(f, words) {
			continue
		}
		if len(matches) == limit {
			return matches, true
		}
		matches = append(matches, f)
	}
	return matches, false
}

// holdsWords reports whether text holds every one of words, which are in
// lower case, ignoring case.
func holdsWords(text string, words []string) bool {
	text = strings.ToLower(text)
	return !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) })
}

// branchQuery is the query of a search of the branches: what follows
// branchPrefix.
func branchQuery(query string) (string, bool) {
	return strings.CutPrefix(strings.TrimSpace(query), branchPrefix)
}

// matchSources keeps the sources whose branch or worktree's directory
// name, together, hold every word of the query, ignoring case.
func matchSources(sources []repo.Source, query string) []repo.Source {
	words := strings.Fields(strings.ToLower(query))
	return slices.DeleteFunc(slices.Clone(sources), func(s repo.Source) bool {
		name := s.Branch
		if s.Worktree != "" {
			name += " " + filepath.Base(s.Worktree)
		}
		return !holdsWords(name, words)
	})
}

// searchSource is the source a Files view searches: the one chosen, else
// the current file's, else the first.
func (a *app) searchSource(r *repoState, st *filesViewState, project string) repo.Source {
	if s, ok := r.source(st.search.source); ok {
		return s
	}
	if ref, ok := a.currentFile(project); ok {
		if s, ok := r.source(ref.Source); ok {
			return s
		}
	}
	return r.sources[0]
}

// fileSearchBar is the search field and, before it, the source it searches
// in, which a menu changes, compact to share a row.
func (a *app) fileSearchBar(c *ui.Context, r *repoState, st *filesViewState, s repo.Source) {
	f := &st.search
	ui.MenuButton(c, sourceName(s), func(m *ui.Menu) {
		for _, o := range r.sources {
			if m.Item(sourceName(o)).Checked(o.Key() == s.Key()).Chosen() {
				f.source = o.Key()
			}
		}
	}).Label(a.text("Search in", "搜索范围")).Height(26).FontSize(theme.Caption).MaxWidth(110).Shrink(1)
	field := ui.SearchField(c, &f.query).Label(a.text("Search files", "搜索文件")).
		Placeholder(a.text("Search files, b: for branches", "搜索文件，b: 搜分支")).Height(26).FontSize(theme.Caption).Grow(1).MinWidth(60)
	if a.focusFileSearch {
		field.Focus()
		a.focusFileSearch = false
	}
	if field.Shortcut(0, ui.KeyDown) {
		f.focusResults = true
	}
	if field.Submitted() {
		f.focusResults, f.pinFirst = true, true
	}
}

// fileSearchResults shows the source's files matching the query, as a tree
// with every directory open, the first file chosen when asked.
func (a *app) fileSearchResults(c *ui.Context, r *repoState, st *filesViewState, s repo.Source, project string) {
	f := &st.search
	// Asked once: with no files to choose, the ask goes.
	focus, pin := f.focusResults, f.pinFirst
	f.focusResults, f.pinFirst = false, false
	l := a.listingFor(r, s)
	if l.err != nil || l.files == nil {
		message, color := a.text("Reading…", "正在读取…"), a.theme.TextMuted
		if l.err != nil {
			message, color = l.err.Error(), a.theme.Danger
		}
		ui.Text(c, message).TextColor(color).Padding(12)
		return
	}
	if key := (searchKey{f.query, s.Key(), l, l.reads}); f.made != key {
		f.made = key
		var matches []string
		matches, f.more = matchFiles(l.files, f.query, fileSearchLimit)
		f.results = index(s.Key(), matches)
		f.outline.Open = ui.Selection[fileNode]{}
		for dir := range f.results {
			f.outline.Open.Add(fileNode{source: s.Key(), path: dir})
		}
		f.selected = -1
		f.outline.List.Selected = &f.selected
	}
	if len(f.results[""]) == 0 {
		ui.Text(c, a.text("No files match", "没有匹配的文件")).TextColor(a.theme.TextMuted).Padding(12)
		return
	}
	children := func(n fileNode) []fileNode { return f.results[n.path] }
	outline := a.fileOutline(c, r, &f.outline, f.results[""], children, project)
	if focus {
		for i := range f.outline.Rows() {
			if children(f.outline.Item(i)) == nil {
				f.selected = i
				a.openFile(project, f.outline.Item(i), pin)
				break
			}
		}
		outline.Focus()
	}
	if f.more {
		ui.Text(c, fmt.Sprintf(a.text("Showing the first %d matches; type more to narrow them", "只显示前 %d 个，继续输入可以缩小范围"), fileSearchLimit)).
			FontSize(theme.Caption).TextColor(a.theme.TextMuted).Padding(6, 12)
	}
}

// branchSearchResults shows the branches matching the query by day, as
// the tree does: a click, a double click or Enter on one, or Enter in the
// field for the first, leaves the search for it in the tree.
func (a *app) branchSearchResults(c *ui.Context, r *repoState, st *filesViewState, query, project string) {
	f := &st.search
	focus, reveal := f.focusResults, f.pinFirst
	f.focusResults, f.pinFirst = false, false
	rows := sourceRows(r.top, matchSources(r.sources, query), time.Now(), a.text)
	if len(rows) == 0 {
		ui.Text(c, a.text("No branches match", "没有匹配的分支")).TextColor(a.theme.TextMuted).Padding(12)
		return
	}
	first := slices.IndexFunc(rows, func(n fileNode) bool { return n.day == "" })
	if reveal {
		st.revealSource(rows[first].source)
		c.Invalidate()
		return
	}
	if f.branches.List.Selected == nil {
		f.branch = -1
		f.branches.List.Selected = &f.branch
	}
	f.branches.List.Header = func(i int) bool { return f.branches.Item(i).day != "" }
	clicked := ""
	list := ui.Outline(c, &f.branches, rows, func(fileNode) []fileNode { return nil }, func(n fileNode) {
		if ui.Box(c).FillWidth().Children(func() { a.fileRow(c, r, n, false, project) }).Clicked() && n.day == "" {
			clicked = n.source
		}
	}).Grow(1).FillWidth()
	if list.Submitted() && f.branch >= 0 && f.branch < f.branches.Rows() {
		clicked = f.branches.Item(f.branch).source
	}
	if clicked != "" {
		st.revealSource(clicked)
		c.Invalidate()
		return
	}
	if focus {
		f.branch = first
		list.Focus()
	}
}

// searchFiles shows the side bar's Files and focuses its search field.
func (a *app) searchFiles() {
	p := a.state.CurrentPerspective()
	if p == nil {
		return
	}
	i := slices.IndexFunc(p.Side.Views, func(v *layout.View) bool { return v.Kind == layout.ViewTree })
	if i < 0 {
		return
	}
	p.SideVisible, p.Side.Active = true, p.Side.Views[i].ID
	a.focusFileSearch = true
	a.layoutChanged()
}
