package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// maxCompletions bounds the directories the palette lists.
const maxCompletions = 50

// paletteItem is a row of the command palette: a command, or a directory
// (path) that it opens.
type paletteItem struct {
	title, path string
	run         func()
}

// openPalette shows the command palette with query typed in it: "~/" to
// open a project, "" for commands.
func (a *app) openPalette(query string) {
	a.palette.open, a.palette.query, a.palette.highlight = true, query, 0
}

// isPath reports whether the palette's query is a directory rather than a
// command.
func isPath(q string) bool {
	return strings.HasPrefix(q, "~") || strings.HasPrefix(q, "/") || strings.HasPrefix(q, ".")
}

// expand makes q, as typed in the palette, absolute: "~" is home, and a
// relative path is under base. A trailing "/" stays, for completions.
func expand(q, home, base string) string {
	if q == "~" || strings.HasPrefix(q, "~/") {
		return home + q[1:]
	}
	if filepath.IsAbs(q) {
		return q
	}
	return base + "/" + q
}

// completions lists the directories beside path whose names begin with its
// last part, ignoring case: all of them after a trailing "/", and hidden
// ones only for a part starting with ".". A directory that does not exist,
// as while its name is typed, has none.
func completions(path string) ([]string, error) {
	slash := strings.LastIndex(path, "/")
	dir, prefix := path[:slash+1], strings.ToLower(path[slash+1:])
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") || !strings.HasPrefix(strings.ToLower(name), prefix) {
			continue
		}
		full := filepath.Join(dir, name)
		ok, err := isDirectory(full, e)
		if err != nil {
			return matches, err
		}
		if ok {
			matches = append(matches, full)
		}
		if len(matches) == maxCompletions {
			break
		}
	}
	return matches, nil
}

// isDirectory reports whether e, at path, is a directory or a link to one;
// a link to nothing is not.
func isDirectory(path string, e fs.DirEntry) (bool, error) {
	if e.Type()&fs.ModeSymlink == 0 {
		return e.IsDir(), nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// tilde shows a path under home with "~".
func tilde(path, home string) string {
	if home != "" && (path == home || strings.HasPrefix(path, home+"/")) {
		return "~" + path[len(home):]
	}
	return path
}

// paletteItems lists what the query matches: the directory it names and
// those it begins, or the commands whose titles contain it.
func (a *app) paletteItems() ([]paletteItem, error) {
	q := strings.TrimSpace(a.palette.query)
	if !isPath(q) {
		q = strings.ToLower(q)
		var items []paletteItem
		for _, item := range a.commands() {
			if strings.Contains(strings.ToLower(item.title), q) {
				items = append(items, item)
			}
		}
		return items, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	// A relative path is under the current project's directory.
	base := home
	if p := a.state.CurrentProject(); p != nil {
		base = p.CWD
	}
	path := expand(q, home, base)
	var items []paletteItem
	dir := filepath.Clean(path)
	if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
		items = append(items, a.directoryItem(dir, home))
	}
	matches, err := completions(path)
	for _, m := range matches {
		if m != dir {
			items = append(items, a.directoryItem(m, home))
		}
	}
	items = append(items, paletteItem{title: a.text("Browse…", "浏览…"), run: a.chooseProject})
	return items, err
}

func (a *app) directoryItem(dir, home string) paletteItem {
	return paletteItem{title: tilde(dir, home), path: dir, run: func() { a.openProjectDirectory(dir) }}
}

// commands are what the palette runs, besides opening directories.
func (a *app) commands() []paletteItem {
	items := []paletteItem{
		{title: a.text("Open Project…", "打开项目…"), run: func() { a.openPalette("~/") }},
		{title: a.text("Browse for Project…", "浏览项目目录…"), run: a.chooseProject},
		{title: a.text("New Terminal", "新建终端"), run: func() {
			if r := a.targetRegion(); r != nil {
				a.newTerminal(r.ID)
			}
		}},
		{title: a.text("New Perspective…", "新建视角…"), run: func() { a.openPerspectiveForm(0) }},
		{title: a.text("Toggle Global", "展开或收起 Global"), run: a.toggleGlobal},
		{title: a.text("Toggle Project Names", "展开或收起项目名称"), run: func() {
			a.state.RailExpanded = !a.state.RailExpanded
			a.layoutChanged()
		}},
	}
	for _, p := range a.state.Projects {
		items = append(items, paletteItem{title: a.text("Project: ", "项目：") + p.Name, run: func() { a.selectProject(p.Name) }})
	}
	if project := a.state.CurrentProject(); project != nil {
		for _, p := range project.Perspectives {
			items = append(items, paletteItem{title: a.text("Perspective: ", "视角：") + p.Name, run: func() { a.selectPerspective(p.ID) }})
		}
	}
	return items
}

// runPaletteItem closes the palette before running item, which can open it
// again.
func (a *app) runPaletteItem(item paletteItem) {
	a.palette.open = false
	item.run()
}

// paletteDialog is the command palette: Up and Down move among its items,
// Enter runs one, and Tab goes into the directory highlighted.
func (a *app) paletteDialog(c *ui.Context) {
	p := &a.palette
	ui.Modal(c, &p.open, func() {
		items, err := a.paletteItems()
		p.highlight = max(min(p.highlight, len(items)-1), 0)
		moved := false
		ui.Column(c).Width(560).Gap(8).Children(func() {
			input := ui.TextInput(c, &p.query).Label(a.text("Command or directory", "命令或目录")).
				Placeholder(a.text("Type a command, or a directory such as ~/", "输入命令，或者 ~/ 这样的目录")).AutoFocus().FillWidth()
			if p.caretEnd {
				n := utf8.RuneCountInString(p.query)
				input.SetTextSelection(n, n)
				p.caretEnd = false
			}
			if input.Changed() {
				p.highlight = 0
			}
			if input.Shortcut(0, ui.KeyDown) && p.highlight < len(items)-1 {
				p.highlight++
				moved = true
			}
			if input.Shortcut(0, ui.KeyUp) && p.highlight > 0 {
				p.highlight--
				moved = true
			}
			if input.Shortcut(0, ui.KeyTab) && p.highlight < len(items) && items[p.highlight].path != "" {
				home, _ := os.UserHomeDir() // Without a home, the path shows in full.
				p.query, p.highlight, p.caretEnd = tilde(items[p.highlight].path, home)+"/", 0, true
				c.Invalidate()
			}
			if input.Submitted() && p.highlight < len(items) {
				a.runPaletteItem(items[p.highlight])
				return
			}
			if err != nil {
				ui.Text(c, err.Error()).TextColor(a.theme.Danger)
			}
			ui.Scroll(c).MaxHeight(360).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Gap(2).Children(func() {
					for i, item := range items {
						row := ui.ButtonBase(c.Key(i)).FillWidth().Height(32).Padding(0, 10).Radius(6).Justify(ui.Start).
							Label(item.title).Children(func() {
							ui.Text(c, item.title).SingleLine().Ellipsis("…")
						}).OnClick(func() { a.runPaletteItem(item) })
						if i == p.highlight {
							row.Background(a.theme.SurfaceHover)
							if moved {
								row.ScrollIntoView()
							}
						}
					}
				})
			})
		})
	})
}
