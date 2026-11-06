package app

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/review"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// noteDraft is the note being written on a view, at its bottom: views have
// one each, so that going to another leaves it as it is.
type noteDraft struct {
	view    layout.ViewID
	project string
	source  string
	anchor  review.Anchor
	text    string
	checked bool   // whether Git told if the text is committed
	focus   bool   // focuses the input once, on opening
	review  string // the review of the note edited
	edit    int    // the note edited; 0 for a new one
}

// noteSpot is what a render holds selected, to note.
type noteSpot struct {
	lines *review.Lines
	quote string
}

// noteHere starts a note on what the active view holds selected.
func (a *app) noteHere() {
	v := activeView(a.targetRegion())
	if v == nil {
		return
	}
	if v.Diff != nil {
		a.noteInDiff(v, nil)
	} else if v.File != nil {
		a.noteInFile(v)
	}
}

// noteInFile starts a note on what a file tab holds selected.
func (a *app) noteInFile(v *layout.View) {
	st, project := a.fileViews[v.ID], a.project(v.ProjectName)
	if st == nil || project == nil || len(st.renders) == 0 {
		return
	}
	noted, ok := st.renders[st.shown].(notedRender)
	s, found := a.repoFor(project.CWD).source(v.File.Source)
	if !ok || !found && !filepath.IsAbs(v.File.Source) {
		return
	}
	spot := noted.spot()
	anchor := review.Anchor{Render: renderKinds[st.kinds[st.shown]].name, Path: v.File.Path, Side: review.SideNew,
		Lines: spot.lines, Quote: spot.quote, Head: s.Commit}
	a.startNote(v, v.File.Source, anchor, found, s)
}

// noteInDiff starts a note on the lines given, else those selected, else
// the caret's, of the file a Diff shows.
func (a *app) noteInDiff(v *layout.View, lines *editor.DiffLineRange) {
	st, project := a.diffs[v.ID], a.project(v.ProjectName)
	if st == nil || project == nil || st.file.state == nil || st.selected < 0 {
		return
	}
	s, found := a.repoFor(project.CWD).source(v.Diff.Source)
	if !found {
		return
	}
	d := st.file.state
	span := diffSpan(d, lines)
	buffer, side := d.New, review.SideNew
	if span.Side == editor.DiffOld {
		buffer, side = d.Old, review.SideOld
	}
	quote := make([]string, 0, span.End-span.Start+1)
	for i := span.Start; i <= span.End; i++ {
		quote = append(quote, buffer.Line(i))
	}
	ref := *v.Diff
	ref.Base, ref.Unified = st.base, false
	anchor := review.Anchor{Diff: &ref, Path: st.file.path, Side: side, Quote: strings.Join(quote, "\n"),
		Lines: &review.Lines{First: span.Start + 1, Last: span.End + 1}, Head: s.Commit}
	a.startNote(v, v.Diff.Source, anchor, side == review.SideNew && ref.Commit == "", s)
}

func diffSpan(d *editor.DiffState, lines *editor.DiffLineRange) editor.DiffLineRange {
	selected, ok := d.SelectedLines()
	if lines != nil && (!ok || selected.Side != lines.Side || lines.Start < selected.Start || lines.Start > selected.End) {
		return *lines
	}
	if ok {
		return selected
	}
	line := d.CursorPos().Line
	return editor.DiffLineRange{Side: d.ActiveSide(), Start: line, End: line}
}

// startNote opens the input for a note; with check, it can be saved once
// Git told whether the file is committed.
func (a *app) startNote(v *layout.View, source string, anchor review.Anchor, check bool, s repo.Source) {
	d := &noteDraft{view: v.ID, project: v.ProjectName, source: source, anchor: anchor, checked: !check, focus: true}
	if !a.openDraft(d) || !check {
		return
	}
	path := anchor.Path
	var uncommitted bool
	var err error
	a.background(func() { uncommitted, err = repo.Uncommitted(s, path) }, func() {
		if a.drafts[d.view] != d {
			return
		}
		if err != nil {
			delete(a.drafts, d.view)
			a.report(err)
			return
		}
		d.anchor.Uncommitted, d.checked = uncommitted, true
	})
}

// editNote opens the input on a note of a review's tab.
func (a *app) editNote(v *layout.View, r review.Review, n review.Note) {
	a.openDraft(&noteDraft{view: v.ID, project: v.ProjectName, source: r.Source, anchor: n.Anchor, text: n.Text,
		checked: true, focus: true, review: r.ID, edit: n.ID})
}

// openDraft makes d the view's draft, unless what is written there stays
// until saved or cancelled: it then reports false, and goes back to that.
func (a *app) openDraft(d *noteDraft) bool {
	if written := a.drafts[d.view]; written != nil && strings.TrimSpace(written.text) != "" {
		written.focus = true
		return false
	}
	a.drafts[d.view] = d
	return true
}

// saveNote adds the note to the source's open review, starting one when
// there is none, or changes the note edited.
func (a *app) saveNote(d *noteDraft) {
	project := a.project(d.project)
	text := strings.TrimSpace(d.text)
	if project == nil || text == "" || !d.checked {
		return
	}
	l := a.reviewsFor(project)
	if !l.read {
		return
	}
	var next review.Review
	if d.edit != 0 {
		r, ok := l.find(d.review)
		if !ok || r.Done() {
			a.closeDraft(d) // deleted or done while the note was being edited
			return
		}
		next = review.EditNote(r, d.edit, text)
	} else {
		r, ok := l.open(d.source)
		if !ok {
			r = review.New(project.CWD, d.source, d.anchor.Head, time.Now())
		}
		next = review.AddNote(r, d.anchor, text)
	}
	if err := a.storeReview(next); err != nil {
		a.report(err)
		return
	}
	a.closeDraft(d)
}

func (a *app) closeDraft(d *noteDraft) {
	delete(a.drafts, d.view)
	if r := a.targetRegion(); r != nil {
		a.focusRegion(r.ID)
	}
}

// noteComposer is the input of a note on this view: a pane under its
// content, with its way out on view; Esc is the input's here, as no
// terminal shows.
func (a *app) noteComposer(c *ui.Context, v *layout.View) {
	d := a.drafts[v.ID]
	if d == nil {
		return
	}
	ui.Box(c).FillWidth().Height(1).Shrink(0).Background(a.theme.Border)
	ui.Column(c.Key("composer")).FillWidth().Shrink(0).Padding(8, 12).Gap(6).Background(a.theme.Surface).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, noteLabel(d.anchor)).FontSize(theme.Caption).Font("monospace").MinWidth(0).SingleLine().Ellipsis("…")
			if d.anchor.Uncommitted {
				ui.Text(c, a.text("uncommitted", "未提交")).FontSize(theme.Caption).TextColor(a.theme.Warning).Shrink(0)
			}
			ui.Spacer(c)
			if ui.Button(c, a.text("Cancel", "取消")+" (Esc)").Height(24).Clicked() {
				a.closeDraft(d)
				return
			}
			if ui.PrimaryButton(c, a.text("Save", "保存")+" (⌘↩)").Height(24).Clicked() {
				a.saveNote(d)
			}
		})
		if a.drafts[v.ID] != d {
			return
		}
		input := ui.TextArea(c, &d.text).Label(a.text("Note", "标注")).FillWidth().Height(72)
		if d.focus {
			input.Focus()
			d.focus = false
		}
		if input.Shortcut(ui.Super, ui.KeyEnter) {
			a.saveNote(d)
		} else if input.Shortcut(0, ui.KeyEscape) {
			a.closeDraft(d)
		}
	})
}

// noteActions are a file or Diff header's: the notes of the source's open
// review, which lead to Reviews, and with add, a way to start one.
func (a *app) noteActions(c *ui.Context, v *layout.View, source string, add func()) {
	project := a.project(v.ProjectName)
	if project == nil {
		return
	}
	if r, ok := a.reviewsFor(project).open(source); ok && len(r.Notes) > 0 {
		count := ui.ButtonBase(c).Height(22).Padding(0, 6).Radius(4).Label(a.text("Reviews", "Reviews"))
		count.Children(func() {
			ui.Text(c, a.noteCountName(len(r.Notes))).FontSize(theme.Caption).TextColor(a.theme.Accent)
		})
		if count.Clicked() {
			a.showReviews()
		}
	}
	if add != nil {
		note := ui.ButtonBase(c).Height(22).Padding(0, 6).Radius(4).Label(a.text("Add Note", "添加标注")).
			Tooltip(a.text("Add Note", "添加标注") + " (" + noteKeys.String() + ")")
		note.Children(func() { ui.Text(c, "+").FontSize(theme.Caption).TextColor(a.theme.TextMuted) })
		if note.Clicked() {
			add()
		}
	}
}
