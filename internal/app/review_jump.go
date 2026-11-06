package app

import (
	"cmp"
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
	render := cmp.Or(j.note.Anchor.Render, sourceKind.name) // a Diff's note, gone from its comparison
	if i := slices.IndexFunc(st.kinds, func(k int) bool { return renderKinds[k].name == render }); i >= 0 {
		st.shown = i
	}
	noted, ok := st.renders[st.shown].(notedRender)
	if !ok {
		delete(a.jumps, v.ID)
		return
	}
	place := review.Locate(st.text, j.note.Anchor)
	if !noted.goToPlace(place, j.note.Anchor.Lines != nil) {
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
	f := &st.file
	if st.selected != row || f.path != j.note.Anchor.Path {
		st.selected = row
		load(row)
		return
	}
	if !f.loading && f.err != nil {
		delete(a.jumps, v.ID)
		a.noteNotice(j, f.err.Error())
	} else if !f.loading && f.note != "" {
		delete(a.jumps, v.ID)
		a.noteNotice(j, f.note)
	}
}

// diffJump goes to the note a Diff view waits for, once its file shows.
func (a *app) diffJump(c *ui.Context, v *layout.View, st *diffViewState) {
	j, waiting := a.jumps[v.ID]
	f := &st.file
	if !waiting || f.path != j.note.Anchor.Path || f.loading {
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
