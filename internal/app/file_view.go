package app

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"unicode/utf8"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// fileLimit is the largest file a file view reads.
const fileLimit = 10 << 20

// fileViewState is what a file view read: the file's text in a read-only
// editor, or why it shows none.
type fileViewState struct {
	ref     layout.FileRef
	read    bool
	source  repo.Source // as it was when read: a new commit or checkout reads again
	version int         // the repository's when read
	loading bool
	editor  *editor.State
	size    int64
	binary  bool
	err     error
}

// fileView shows a file of a source of the project's repository, read
// again as the worktree it is read from changes.
func (a *app) fileView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil || v.File == nil {
		return
	}
	r := a.repoFor(project.CWD)
	st := a.fileViews[v.ID]
	if st == nil || st.ref != *v.File {
		st = &fileViewState{ref: *v.File}
		a.fileViews[v.ID] = st
	}
	s, found := r.source(v.File.Source)
	if found && !st.loading && (!st.read || st.source != s || s.Worktree != "" && st.version != r.version) {
		a.readFile(r, s, st)
	}
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		ui.Row(c).FillWidth().Height(28).Padding(0, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, v.File.Path).FontSize(theme.Caption).MinWidth(0).SingleLine().Ellipsis("…").Shrink(1)
			if found {
				ui.Text(c, sourceName(s)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine()
			}
			if st.read && st.err == nil {
				ui.Text(c, byteSize(st.size)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine()
			}
		})
		if !found && r.sources != nil {
			a.fileProblem(c, r, project.CWD, st, fmt.Errorf("%s: %s", a.text("No longer in the repository", "仓库里已经没有它"), v.File.Source), s)
			return
		}
		if st.err != nil {
			a.fileProblem(c, r, project.CWD, st, st.err, s)
			return
		}
		if st.editor != nil {
			editor.View(c, st.editor).Grow(1).FillWidth()
			return
		}
		if st.read {
			a.fileProblem(c, r, project.CWD, st, nil, s)
		}
	})
}

// readFile reads the file in the background: text into the editor, kept
// with its place as the text changes, else whether it is binary or too
// large.
func (a *app) readFile(r *repoState, s repo.Source, st *fileViewState) {
	st.loading = true
	top, version, ref := r.top, r.version, st.ref
	var data []byte
	var size int64
	var err error
	a.background(func() { data, size, err = repo.Read(top, s, ref.Path, fileLimit) }, func() {
		st.loading, st.read, st.source, st.version, st.size, st.err = false, true, s, version, size, err
		st.binary = bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data)
		if err != nil || size > fileLimit || st.binary {
			st.editor = nil
			return
		}
		if st.editor == nil {
			buffer := editor.NewBuffer(string(data))
			buffer.SetLanguage(editor.LanguageFor(path.Base(ref.Path)))
			st.editor = &editor.State{Buffer: buffer, ReadOnly: true, Label: ref.Path}
		} else if text := string(data); st.editor.Buffer.String() != text {
			st.editor.Buffer.Reset(text)
		}
	})
}

// fileProblem says why a file view shows no text: an error, with a retry,
// or a binary or too large file, which a worktree's default app can open.
func (a *app) fileProblem(c *ui.Context, r *repoState, dir string, st *fileViewState, err error, s repo.Source) {
	ui.Column(c).Padding(28).Gap(12).FillWidth().AlignItems(ui.Start).Children(func() {
		if err != nil {
			ui.Text(c, err.Error()).TextColor(a.theme.Danger)
			if ui.Button(c, a.text("Read Again", "重新读取")).Clicked() {
				st.read = false
				a.refreshRepo(r, dir)
			}
			return
		}
		message := a.text("Too large to show", "文件太大，不显示")
		if st.binary {
			message = a.text("Binary file", "二进制文件")
		}
		ui.Text(c, message).TextColor(a.theme.TextMuted)
		if s.Worktree != "" && ui.Button(c, a.text("Open with Default App", "用默认应用打开")).Clicked() {
			if err := mygo.Shell.OpenPath(filepath.Join(s.Worktree, filepath.FromSlash(st.ref.Path))); err != nil {
				a.report(err)
			}
		}
	})
}

// byteSize is a size as people read it: 512 B, 3.4 KB, 12 MB.
func byteSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB"}
	size, i := float64(n)/1024, 0
	for size >= 1024 && i < len(units)-1 {
		size, i = size/1024, i+1
	}
	if size < 10 {
		return fmt.Sprintf("%.1f %s", size, units[i])
	}
	return fmt.Sprintf("%.0f %s", size, units[i])
}
