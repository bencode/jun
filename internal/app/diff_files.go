package app

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// diffRow is a row of a Diff view's files: a directory's heading, or a
// file changed.
type diffRow struct {
	dir    string // set for a heading
	change repo.Change
}

// diffFile is what a Diff view read of its chosen file.
type diffFile struct {
	path    string
	loads   int
	loading bool
	err     error
	note    string            // why the file shows no lines: binary, or too large
	state   *editor.DiffState // kept as the file is read again, so that it keeps its place
}

// diffRows lists the files changed under their directories' headings,
// the top level's first, without one.
func diffRows(changes []repo.Change) []diffRow {
	sorted := slices.Clone(changes)
	slices.SortFunc(sorted, func(x, y repo.Change) int {
		return cmp.Or(strings.Compare(path.Dir(x.Path), path.Dir(y.Path)), strings.Compare(x.Path, y.Path))
	})
	var rows []diffRow
	heading := "."
	for _, ch := range sorted {
		if dir := path.Dir(ch.Path); dir != heading {
			heading = dir
			rows = append(rows, diffRow{dir: dir})
		}
		rows = append(rows, diffRow{change: ch})
	}
	return rows
}

// chooseDiffRow is the row of the file at name, else the first file's;
// -1 when there is none.
func chooseDiffRow(rows []diffRow, name string) int {
	if i := slices.IndexFunc(rows, func(r diffRow) bool { return r.dir == "" && r.change.Path == name }); i >= 0 {
		return i
	}
	return slices.IndexFunc(rows, func(r diffRow) bool { return r.dir == "" })
}

func (a *app) diffFileList(c *ui.Context, st *diffViewState, r *repoState, s repo.Source) {
	rows := st.rows
	st.list.Key = func(i int) any {
		if rows[i].dir != "" {
			return "dir:" + rows[i].dir
		}
		return rows[i].change.Path
	}
	st.list.Header = func(i int) bool { return rows[i].dir != "" }
	st.list.Label = func(i int) string { return cmp.Or(rows[i].dir, rows[i].change.Path) }
	list := ui.List(c, &st.list, len(rows), func(i int) {
		if rows[i].dir != "" {
			ui.Text(c, rows[i].dir+"/").FontSize(theme.Caption).TextColor(a.theme.TextMuted).
				FillWidth().SingleLine().Ellipsis("…").Padding(6, 10, 2).Background(a.theme.Background)
			return
		}
		a.diffFileRow(c, rows[i].change, path.Dir(rows[i].change.Path) != ".")
	}).Width(260).Shrink(0).FillHeight()
	if list.Changed() && st.selected >= 0 && st.selected < len(rows) && rows[st.selected].dir == "" {
		a.loadDiffFile(st, r, s, rows[st.selected].change)
	}
}

func (a *app) diffFileRow(c *ui.Context, ch repo.Change, nested bool) {
	indent := float32(10)
	if nested {
		indent = 20
	}
	ui.Row(c).FillWidth().Height(26).Padding(0, 10, 0, indent).Gap(6).AlignItems(ui.Center).Children(func() {
		a.diffStatus(c, ch.Status)
		ui.Text(c, path.Base(ch.Path)).MinWidth(0).SingleLine().Ellipsis("…")
		if ch.OldPath != "" {
			ui.Text(c, "← "+path.Base(ch.OldPath)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).MinWidth(0).Shrink(100).SingleLine().Ellipsis("…")
		}
		ui.Spacer(c)
		a.diffLines(c, ch)
	})
}

// diffStatus is a change's letter, colored as its lines are.
func (a *app) diffStatus(c *ui.Context, status byte) {
	color := map[byte]ui.Color{'A': a.theme.Success, 'M': a.theme.Warning, 'D': a.theme.Danger, 'R': a.theme.Accent}[status]
	ui.Text(c, string(status)).FontSize(theme.Caption).Font("monospace").Bold().TextColor(color).Width(12).Shrink(0)
}

func (a *app) diffLines(c *ui.Context, ch repo.Change) {
	if ch.Added < 0 {
		ui.Text(c, a.text("binary", "二进制")).FontSize(theme.Caption).TextColor(a.theme.TextMuted).Shrink(0)
		return
	}
	if ch.Added > 0 {
		ui.Text(c, fmt.Sprintf("+%d", ch.Added)).FontSize(theme.Caption).TextColor(a.theme.Success).Shrink(0)
	}
	if ch.Deleted > 0 {
		ui.Text(c, fmt.Sprintf("−%d", ch.Deleted)).FontSize(theme.Caption).TextColor(a.theme.Danger).Shrink(0)
	}
}

// loadDiffFile reads the file's two sides off the UI thread: as it was at
// the diff's start, and as it is at its end.
func (a *app) loadDiffFile(st *diffViewState, r *repoState, s repo.Source, ch repo.Change) {
	f := &st.file
	if f.path != ch.Path {
		*f = diffFile{path: ch.Path, loads: f.loads}
	}
	f.loads++
	load := f.loads
	f.loading = true
	top, d := r.top, st.diff
	tooLarge := a.text("The file is over 10 MB; its changes do not show.", "文件超过 10 MB，不显示差异。")
	binary := a.text("A binary file; its changes do not show.", "二进制文件，不显示差异。")
	var before, after []byte
	var note string
	var err error
	a.background(func() {
		read := func(read func() ([]byte, int64, error)) []byte {
			if err != nil || note != "" {
				return nil
			}
			data, size, readErr := read()
			if err = readErr; err == nil && size > fileLimit {
				note = tooLarge
			} else if err == nil && !isText(data) {
				note = binary
			}
			return data
		}
		if ch.Status != 'A' {
			before = read(func() ([]byte, int64, error) { return repo.ReadAt(top, d.From, cmp.Or(ch.OldPath, ch.Path), fileLimit) })
		}
		if ch.Status != 'D' && d.To == "" {
			after = read(func() ([]byte, int64, error) { return repo.Read(top, s, ch.Path, fileLimit) })
		} else if ch.Status != 'D' {
			after = read(func() ([]byte, int64, error) { return repo.ReadAt(top, d.To, ch.Path, fileLimit) })
		}
	}, func() {
		if load != f.loads {
			return
		}
		f.loading, f.err, f.note = false, err, note
		if err != nil || note != "" {
			return
		}
		if f.state == nil {
			language := editor.LanguageFor(path.Base(ch.Path))
			old, current := editor.NewBuffer(string(before)), editor.NewBuffer(string(after))
			old.SetLanguage(language)
			current.SetLanguage(language)
			f.state = &editor.DiffState{Old: old, New: current}
			return
		}
		for buffer, text := range map[*editor.Buffer]string{f.state.Old: string(before), f.state.New: string(after)} {
			if buffer.String() != text {
				buffer.Reset(text)
			}
		}
	})
}

// diffFileView shows the chosen file's changes, side by side or unified.
func (a *app) diffFileView(c *ui.Context, st *diffViewState, unified bool) {
	f := &st.file
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Gap(0).Children(func() {
		if st.selected < 0 || st.selected >= len(st.rows) || st.rows[st.selected].dir != "" {
			return
		}
		ch := st.rows[st.selected].change
		ui.Row(c).FillWidth().Height(32).Shrink(0).Padding(0, 14).Gap(8).AlignItems(ui.Center).Background(a.theme.Surface).Children(func() {
			a.diffStatus(c, ch.Status)
			name := ch.Path
			if ch.OldPath != "" {
				name = ch.OldPath + " → " + ch.Path
			}
			ui.Text(c, name).FontSize(theme.Caption).Font("monospace").MinWidth(0).SingleLine().Ellipsis("…").Selectable()
			ui.Spacer(c)
			a.diffLines(c, ch)
		})
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(a.theme.Border)
		if f.path != ch.Path {
			return
		}
		if f.err != nil || f.note != "" {
			message, color := f.note, a.theme.TextMuted
			if f.err != nil {
				message, color = f.err.Error(), a.theme.Danger
			}
			ui.Text(c, message).TextColor(color).Padding(16)
			return
		}
		if f.state == nil {
			return
		}
		f.state.Layout = editor.DiffSplit
		if unified {
			f.state.Layout = editor.DiffUnified
		}
		f.state.Font.Size = a.textSize() // in the system's monospaced font, as Family is empty
		editor.DiffView(c, f.state).Grow(1).FillWidth()
	})
}
