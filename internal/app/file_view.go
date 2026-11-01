package app

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// fileLimit is the largest file a file view reads.
const fileLimit = 10 << 20

// fileViewState is what a file view read: the renders that can show the
// file, and the one that does, or why none can.
type fileViewState struct {
	ref     layout.FileRef
	read    bool
	source  repo.Source // as it was when read: a new commit or checkout reads again
	version int         // the repository's when read
	loading bool
	size    int64
	err     error
	kinds   []int        // the renderKinds that can show the file
	renders []fileRender // one for each of kinds
	shown   int          // the index in renders of the one showing
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
		a.readFile(fileEnv{a, r, s, *v.File, v.ProjectName}, st)
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
			if len(st.renders) > 0 {
				if detail := st.renders[st.shown].detail(); detail != "" {
					ui.Text(c, detail).FontSize(theme.Caption).TextColor(a.theme.TextMuted).SingleLine()
				}
			}
			if len(st.kinds) > 1 {
				ui.Spacer(c)
				labels := make([]string, len(st.kinds))
				for i, k := range st.kinds {
					labels[i] = renderKinds[k].label(a)
				}
				ui.Segmented(c, &st.shown, labels...).Label(a.text("Show as", "显示方式")).Height(22).FontSize(theme.Caption)
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
		if len(st.renders) > 0 {
			st.renders[st.shown].view(c, a.textSize())
			return
		}
		if st.read {
			a.fileProblem(c, r, project.CWD, st, nil, s)
		}
	})
}

// readFile reads the file in the background, then gives its content to
// the renders that can show it: those it had, keeping their state, unless
// they differ, or the source moved on.
func (a *app) readFile(f fileEnv, st *fileViewState) {
	st.loading = true
	top, version := f.r.top, f.r.version
	var data []byte
	var size int64
	var err error
	a.background(func() { data, size, err = repo.Read(top, f.s, f.ref.Path, fileLimit) }, func() {
		moved := st.source != f.s
		st.loading, st.read, st.source, st.version, st.size, st.err = false, true, f.s, version, size, err
		kinds := []int{}
		if err == nil && size <= fileLimit {
			kinds = kindsFor(f.ref.Path, data)
		}
		if moved || !slices.Equal(kinds, st.kinds) {
			if !slices.Equal(kinds, st.kinds) {
				st.shown = 0
			}
			st.kinds, st.renders = kinds, make([]fileRender, len(kinds))
			for i, k := range kinds {
				st.renders[i] = renderKinds[k].make(f)
			}
		}
		for _, render := range st.renders {
			render.update(data)
		}
	})
}

// fileProblem says why a file view shows nothing: an error, with a retry,
// or a file too large or no render shows, which a worktree's default app
// can open.
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
		message := a.text("Binary file", "二进制文件")
		if st.size > fileLimit {
			message = a.text("Too large to show", "文件太大，不显示")
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
