package app

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// diffCommitsLimit is how many of a branch's commits the scope offers.
const diffCommitsLimit = 50

var splitIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M12 4v16"/></svg>`))

// diffViewState is what a Diff view read of its comparison.
type diffViewState struct {
	read     layout.DiffRef // what was read, Unified cleared: another is read anew
	version  int            // the repository's version when read; -1 to read again
	loads    int            // counts reads, so that a later one's results replace an earlier one's
	loading  bool
	done     bool // read once
	err      error
	base     string // compared to: the ref's, else the guess
	guessed  string // guessed once, as it takes a git command a branch
	bases    []string
	commits  []repo.Commit
	diff     repo.Diff
	rows     []diffRow
	selected int
	list     ui.ListState
	file     diffFile
}

// diffView shows a source's changes: the files changed, and the chosen
// one's lines, read again as the worktree changes.
func (a *app) diffView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil || v.Diff == nil {
		return
	}
	r := a.repoFor(project.CWD)
	ref := *v.Diff
	ref.Unified = false
	st := a.diffs[v.ID]
	if st == nil || st.read != ref {
		previous := st
		st = &diffViewState{read: ref, version: -1, selected: -1}
		st.list.Selected = &st.selected
		if previous != nil && previous.read.Source == ref.Source {
			st.guessed = previous.guessed
		}
		a.diffs[v.ID] = st
	}
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		if r.sources == nil {
			a.filesMessage(c, r, project.CWD)
			return
		}
		s, ok := r.source(ref.Source)
		if !ok {
			ui.Text(c, fmt.Sprintf(a.text("%s no longer exists.", "%s 已不存在。"), ref.Source)).TextColor(a.theme.TextMuted).Padding(16)
			return
		}
		if s.Worktree != "" {
			a.watch(r, s.Worktree)
		}
		if !st.loading && st.version != r.version {
			a.readDiff(st, r, s, ref)
		}
		a.diffHeader(c, v, st, s)
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(a.theme.Border)
		a.diffBody(c, v, st, r, s)
	})
}

func (a *app) diffBody(c *ui.Context, v *layout.View, st *diffViewState, r *repoState, s repo.Source) {
	if st.err != nil {
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			ui.Text(c, st.err.Error()).TextColor(a.theme.Danger)
			if ui.Button(c, a.text("Read Again", "重新读取")).Clicked() {
				st.version = -1
			}
		})
		return
	}
	if !st.done || len(st.rows) == 0 {
		message := a.text("No changes.", "没有改动。")
		if !st.done {
			message = a.text("Reading…", "正在读取…")
		}
		ui.Text(c, message).TextColor(a.theme.TextMuted).Padding(16)
		return
	}
	ui.Row(c).Grow(1).FillWidth().MinHeight(0).Gap(0).AlignItems(ui.Stretch).Children(func() {
		a.diffFileList(c, st, r, s)
		ui.Box(c).Width(1).FillHeight().Shrink(0).Background(a.theme.Border)
		a.diffFileView(c, st, v.Diff.Unified)
	})
}

// readDiff reads the comparison off the UI thread: what it can compare to,
// the base, the changes and the branch's commits.
func (a *app) readDiff(st *diffViewState, r *repoState, s repo.Source, ref layout.DiffRef) {
	st.loads++
	load := st.loads
	st.loading, st.version = true, r.version
	top, guessed := r.top, st.guessed
	noBase := a.text("There is no other branch to compare to.", "没有可以对比的其他分支。") // the language is read on the UI thread
	var bases []string
	var commits []repo.Commit
	var diff repo.Diff
	var err error
	a.background(func() {
		if bases, err = repo.Bases(top); err != nil {
			return
		}
		if ref.Base == "" && guessed == "" {
			if guessed, err = repo.GuessBase(top, s, bases); err != nil {
				return
			}
		}
		base := cmp.Or(ref.Base, guessed)
		if base != "" {
			if commits, err = repo.Commits(top, s, base, diffCommitsLimit); err != nil {
				return
			}
		} else if !ref.Uncommitted && ref.Commit == "" {
			err = errors.New(noBase)
			return
		}
		diff, err = repo.Changes(top, s, repo.Compare{Base: base, Uncommitted: ref.Uncommitted, Commit: ref.Commit})
	}, func() {
		if load != st.loads {
			return
		}
		st.loading, st.err = false, err
		if err != nil {
			return
		}
		st.done, st.guessed, st.bases, st.commits, st.diff = true, guessed, bases, commits, diff
		st.base = cmp.Or(ref.Base, guessed)
		st.rows = diffRows(diff.Changes)
		st.selected = chooseDiffRow(st.rows, st.file.path)
		if st.selected >= 0 {
			a.loadDiffFile(st, r, s, st.rows[st.selected].change)
		}
	})
}

// diffHeader names the source, chooses what it is compared to and which
// of its changes show, counts them, and switches the layout.
func (a *app) diffHeader(c *ui.Context, v *layout.View, st *diffViewState, s repo.Source) {
	ref := v.Diff
	changed := func() { a.layoutChanged() }
	ui.Row(c).FillWidth().Height(44).Shrink(0).Padding(0, 10, 0, 14).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, sourceName(s)).Bold().MinWidth(0).SingleLine().Ellipsis("…")
		a.diffPicker(c, a.text("Compare to", "对比"), cmp.Or(st.base, "—"), func(m *ui.Menu) {
			for _, b := range st.bases {
				name := b
				if b == st.guessed {
					name += a.text(" (guessed)", "（推测）")
				}
				if m.Item(name).Radio(b == st.base).Chosen() {
					ref.Base = b
					changed()
				}
			}
		})
		a.diffPicker(c, a.text("Scope", "范围"), a.diffScopeName(ref), func(m *ui.Menu) {
			if m.Item(a.text("All Changes", "全部改动")).Radio(!ref.Uncommitted && ref.Commit == "").Chosen() {
				ref.Uncommitted, ref.Commit = false, ""
				changed()
			}
			if s.Worktree != "" && m.Item(a.text("Uncommitted Only", "只看未提交")).Radio(ref.Uncommitted).Chosen() {
				ref.Uncommitted, ref.Commit = true, ""
				changed()
			}
			m.Submenu(a.text("One Commit", "单个提交"), func(m *ui.Menu) {
				if len(st.commits) == 0 {
					m.Item(a.text("No commits since the base", "基准之后没有提交")).Disabled(true)
				}
				for _, commit := range st.commits {
					if m.Item(commit.Hash + "  " + commit.Subject).Radio(ref.Commit == commit.Hash).Chosen() {
						ref.Uncommitted, ref.Commit = false, commit.Hash
						changed()
					}
				}
			})
		})
		ui.Spacer(c)
		a.diffStats(c, st.diff.Changes)
		toggle := iconButton(c, splitIcon, a.text("Side by Side or Unified", "并排或合并"), 28, 15)
		if !ref.Unified {
			toggle.Background(a.theme.SurfaceHover)
		}
		if toggle.Clicked() {
			ref.Unified = !ref.Unified
			changed()
		}
		if iconButton(c, refreshIcon, a.text("Read Again", "重新读取"), 28, 15).Clicked() {
			st.version = -1
		}
	})
}

// diffPicker is a labeled button choosing from a menu.
func (a *app) diffPicker(c *ui.Context, label, value string, build func(m *ui.Menu)) {
	ui.Text(c, label).FontSize(theme.Caption).TextColor(a.theme.TextMuted).Shrink(0)
	ui.ButtonBase(c).Height(26).Padding(0, 8, 0, 10).Gap(6).Radius(6).Border(1, a.theme.Border).Background(a.theme.Surface).
		MinWidth(0).Label(label).Children(func() {
		ui.Text(c, value).FontSize(theme.Caption).MinWidth(0).SingleLine().Ellipsis("…")
		ui.Icon(c, downIcon).Size(12, 12).TextColor(a.theme.TextMuted)
	}).Menu(build)
}

func (a *app) diffScopeName(ref *layout.DiffRef) string {
	if ref.Commit != "" {
		return ref.Commit
	}
	if ref.Uncommitted {
		return a.text("Uncommitted Only", "只看未提交")
	}
	return a.text("All Changes", "全部改动")
}

// diffStats counts the files changed and their lines added and deleted.
func (a *app) diffStats(c *ui.Context, changes []repo.Change) {
	added, deleted := 0, 0
	for _, ch := range changes {
		added += max(ch.Added, 0)
		deleted += max(ch.Deleted, 0)
	}
	ui.Text(c, fmt.Sprintf(a.text("%d files", "%d 个文件"), len(changes))).FontSize(theme.Caption).TextColor(a.theme.TextMuted).Shrink(0)
	ui.Text(c, fmt.Sprintf("+%d", added)).FontSize(theme.Caption).TextColor(a.theme.Success).Shrink(0)
	ui.Text(c, fmt.Sprintf("−%d", deleted)).FontSize(theme.Caption).TextColor(a.theme.Danger).Shrink(0)
}
