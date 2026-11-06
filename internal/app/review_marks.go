package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/review"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// noteMark is a note and where it is in the text showing.
type noteMark struct {
	note  review.Note
	lines review.Lines
}

// noteMarks places the notes on a side of the file in its text, leaving
// out those with no line and those whose text is gone.
func noteMarks(notes []review.Note, path string, side review.Side, text string) []noteMark {
	var marks []noteMark
	for _, n := range notes {
		if n.Anchor.Path != path || n.Anchor.Side != side || n.Anchor.Lines == nil {
			continue
		}
		if place := review.Locate(text, n.Anchor); place.Status != review.Lost {
			marks = append(marks, noteMark{n, place.Lines})
		}
	}
	return marks
}

// showFileMarks places the open review's notes in the file, again as the
// review or the file changes.
func (a *app) showFileMarks(v *layout.View, project *layout.Project, st *fileViewState) {
	key := markKey{reviews: a.reviewsChanged, text: st.reads}
	if st.marked == key {
		return
	}
	st.marked = key
	r, _ := a.reviewsFor(project).open(v.File.Source)
	marks := noteMarks(r.Notes, v.File.Path, review.SideNew, st.text)
	for _, render := range st.renders {
		if noted, ok := render.(notedRender); ok {
			noted.showMarks(marks)
		}
	}
}

// showDiffMarks shows the open review's notes under their lines, again
// as the review or the file changes.
func (a *app) showDiffMarks(v *layout.View, f *diffFile) {
	d := f.state
	key := markKey{a.reviewsChanged, d.Old.Version() + d.New.Version(), a.textSize()}
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
	size    float32 // a Diff's text size, which its notes' heights follow
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
