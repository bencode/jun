package app

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/review"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// noteJump is a note whose place a view is to show once it has read it.
type noteJump struct {
	project string
	review  string
	source  string
	note    review.Note
}

func noteKey(id string, note int) string { return fmt.Sprintf("%s:%d", id, note) }

// jumpNote shows again what the view showed as the note was made, for the
// view to go to its place when it has read it.
func (a *app) jumpNote(project *layout.Project, r review.Review, n review.Note) {
	j := noteJump{project.Name, r.ID, r.Source, n}
	delete(a.reviewNotices, noteKey(r.ID, n.ID))
	rs := a.repoFor(project.CWD)
	if _, found := rs.source(r.Source); !found && rs.sources != nil && !filepath.IsAbs(r.Source) {
		a.noteNotice(j, a.text("The branch no longer exists.", "分支已不存在。"))
		return
	}
	if n.Anchor.Diff == nil {
		a.jumpFile(j)
		return
	}
	target := a.tabsTarget()
	if target == nil {
		return
	}
	id, err := a.state.ShowDiff(target.ID, project.Name, *n.Anchor.Diff)
	if err != nil {
		a.report(err)
		return
	}
	a.jumps[id] = j
	a.layoutChanged()
	a.focusRegion(target.ID)
}

// jumpFile goes to the note in a file tab.
func (a *app) jumpFile(j noteJump) {
	target := a.tabsTarget()
	if target == nil {
		return
	}
	a.openFile(j.project, fileNode{source: j.source, path: j.note.Anchor.Path}, true)
	if v := activeView(target); v != nil {
		a.jumps[v.ID] = j
	}
	a.focusRegion(target.ID)
}

func (a *app) noteNotice(j noteJump, text string) {
	a.reviewNotices[noteKey(j.review, j.note.ID)] = text
}

// noteFound tells on the note's row how its place was found.
func (a *app) noteFound(j noteJump, place review.Place) {
	if place.Status == review.Moved {
		a.noteNotice(j, fmt.Sprintf(a.text("Moved to line %d.", "已移到第 %d 行。"), place.Lines.First))
	}
	if place.Status == review.Lost {
		a.noteNotice(j, a.text("The text has changed.", "原文已变化。"))
	}
}

// fileJump goes to the note a file view waits for, once it read the file.
func (a *app) fileJump(c *ui.Context, v *layout.View, st *fileViewState) {
	j, waiting := a.jumps[v.ID]
	if !waiting {
		return
	}
	if st.err != nil || st.read && !st.loading && len(st.renders) == 0 {
		delete(a.jumps, v.ID)
		a.noteNotice(j, a.text("The file no longer exists.", "文件已不存在。"))
		return
	}
	if !st.read || st.loading {
		return
	}
	if i := slices.IndexFunc(st.kinds, func(k int) bool { return renderKinds[k].name == j.note.Anchor.Render }); i >= 0 {
		st.shown = i
	}
	place := review.Locate(st.text, j.note.Anchor)
	if !st.renders[st.shown].goToPlace(place, j.note.Anchor.Lines != nil) {
		c.Invalidate() // it lays out first
		return
	}
	delete(a.jumps, v.ID)
	a.noteFound(j, place)
}

// diffJumpFile chooses the file of the note a Diff view waits for, or goes
// to a file tab when the comparison no longer changes that file.
func (a *app) diffJumpFile(v *layout.View, st *diffViewState, load func(row int)) {
	j, waiting := a.jumps[v.ID]
	if waiting && st.err != nil {
		delete(a.jumps, v.ID)
		a.noteNotice(j, st.err.Error())
		return
	}
	if !waiting || !st.done || st.loading {
		return
	}
	row := slices.IndexFunc(st.rows, func(r diffRow) bool { return r.dir == "" && r.change.Path == j.note.Anchor.Path })
	if row < 0 {
		delete(a.jumps, v.ID)
		a.jumpFile(j)
		return
	}
	if st.selected != row || st.file.path != j.note.Anchor.Path {
		st.selected = row
		load(row)
	}
}

// diffJump goes to the note a Diff view waits for, once its file shows.
func (a *app) diffJump(c *ui.Context, v *layout.View, st *diffViewState) {
	j, waiting := a.jumps[v.ID]
	f := &st.file
	if !waiting || f.path != j.note.Anchor.Path || f.loading {
		return
	}
	if f.err != nil || f.note != "" {
		delete(a.jumps, v.ID)
		a.noteNotice(j, a.text("The file no longer shows here.", "这里已看不到这个文件。"))
		return
	}
	d := f.state
	buffer, side := d.New, editor.DiffNew
	if j.note.Anchor.Side == review.SideOld {
		buffer, side = d.Old, editor.DiffOld
	}
	place := review.Locate(buffer.String(), j.note.Anchor)
	top := d.LineTop(side, place.Lines.First-1)
	if d.ViewportHeight() == 0 {
		c.Invalidate() // it lays out first
		return
	}
	d.SetSelectedLines(editor.DiffLineRange{Side: side, Start: place.Lines.First - 1, End: place.Lines.Last - 1}, true)
	d.SetScrollY(max(top-d.ViewportHeight()/3, 0))
	delete(a.jumps, v.ID)
	a.noteFound(j, place)
}

// showDiffMarks shows the open review's notes under their lines, again
// as the review or the file changes.
func (a *app) showDiffMarks(v *layout.View, f *diffFile) {
	d := f.state
	key := markKey{a.reviewsChanged, d.Old.Version() + d.New.Version()}
	if f.marked == key {
		return
	}
	f.marked = key
	project := a.project(v.ProjectName)
	if project == nil {
		return
	}
	r, _ := a.reviewsFor(project).open(v.Diff.Source)
	f.marks = slices.Concat(noteMarks(r.Notes, f.path, review.SideNew, d.New.String()), noteMarks(r.Notes, f.path, review.SideOld, d.Old.String()))
	d.Annotations = nil
	for _, m := range f.marks {
		side := editor.DiffNew
		if m.note.Anchor.Side == review.SideOld {
			side = editor.DiffOld
		}
		d.Annotations = append(d.Annotations, editor.DiffAnnotation{Side: side, Line: m.lines.Last - 1, Key: m.note.ID,
			Height: 12 + float32(len(firstLines(m.note.Text, quoteLines)))*a.textSize()*1.5})
	}
	d.RenderAnnotation = func(c *ui.Context, annotation editor.DiffAnnotation) { a.diffNote(c, v, r, annotation) }
}

// markKey is what a view's marks were placed for.
type markKey struct {
	reviews int
	text    uint64
}

// diffNote shows a note under its line; a click opens its review.
func (a *app) diffNote(c *ui.Context, v *layout.View, r review.Review, annotation editor.DiffAnnotation) {
	i := slices.IndexFunc(r.Notes, func(n review.Note) bool { return n.ID == annotation.Key })
	if i < 0 {
		return
	}
	note := ui.ButtonBase(c).Fill().Padding(6, 12).Background(a.theme.Surface).Label(r.Notes[i].Text)
	note.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(0).AlignItems(ui.Start).Children(func() {
			for _, line := range firstLines(r.Notes[i].Text, quoteLines) {
				ui.Text(c, line).MinWidth(0).SingleLine().Ellipsis("…")
			}
		})
	})
	if note.Clicked() {
		a.openReview(v.ProjectName, r.ID, false)
	}
}
