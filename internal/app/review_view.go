package app

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/review"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo/ui"
)

// quoteLines is how much of a note's text the review shows.
const quoteLines = 3

// reviewView shows a review: what it was of, and its notes, each leading
// to where it was made. Done, it is read-only.
func (a *app) reviewView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil {
		return
	}
	l := a.reviewsFor(project)
	r, ok := l.find(v.Review)
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		if !ok {
			if l.read {
				ui.Text(c, a.text("This review has been deleted.", "这条 Review 已删除。")).TextColor(a.theme.TextMuted).Padding(16)
			}
			return
		}
		a.reviewHeader(c, r)
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(a.theme.Border)
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(4, 0).Gap(0).Children(func() {
				if len(r.Notes) == 0 {
					ui.Text(c, a.text("No notes left.", "没有标注了。")).TextColor(a.theme.TextMuted).Padding(12)
				}
				for _, n := range r.Notes {
					c.Key(n.ID)
					a.reviewNote(c, v, project, r, n)
				}
			})
		})
		a.noteComposer(c, v)
	})
}

func (a *app) reviewHeader(c *ui.Context, r review.Review) {
	ui.Row(c).FillWidth().Height(44).Shrink(0).Padding(0, 10, 0, 14).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, diffSourceName(r.Source)).Bold().MinWidth(0).SingleLine().Ellipsis("…")
		ui.Text(c, a.reviewDetail(r)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).MinWidth(0).Shrink(100).SingleLine().Ellipsis("…")
		if r.Done() {
			ui.Text(c, "✓ "+fmt.Sprintf(a.text("Done %s", "完成于 %s"), r.DoneAt.Local().Format("15:04"))).
				FontSize(theme.Caption).TextColor(a.theme.Success).Shrink(0)
		}
		ui.Spacer(c)
		if ui.Button(c, a.text("Copy Address", "复制地址")).Height(26).Clicked() {
			a.copyReviewAddress(r)
		}
		if !r.Done() && len(r.Notes) > 0 && ui.PrimaryButton(c, a.text("Done", "完成")).Height(26).Clicked() {
			a.finishReview(r)
		}
	})
}

func (a *app) finishReview(r review.Review) {
	done, err := review.Finish(r, time.Now())
	if err == nil {
		err = a.storeReview(done)
	}
	a.report(err)
}

// reviewNote shows a note, which a click follows to its place.
func (a *app) reviewNote(c *ui.Context, v *layout.View, project *layout.Project, r review.Review, n review.Note) {
	ui.Row(c).FillWidth().Gap(4).Padding(2, 8).AlignItems(ui.Start).Children(func() {
		place := ui.ButtonBase(c).Grow(1).MinWidth(0).Padding(6).Radius(6).Label(noteLabel(n.Anchor))
		place.Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Start).Children(func() {
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, noteLabel(n.Anchor)).FontSize(theme.Caption).Font("monospace").MinWidth(0).SingleLine().Ellipsis("…")
					ui.Text(c, a.noteScope(n.Anchor)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).Shrink(0)
				})
				for _, line := range firstLines(n.Anchor.Quote, quoteLines) {
					ui.Text(c, "> "+line).FontSize(theme.Caption).Font("monospace").TextColor(a.theme.TextMuted).
						MinWidth(0).SingleLine().Ellipsis("…")
				}
				ui.Text(c, n.Text).FillWidth()
				if notice := a.reviewNotices[noteKey(r.ID, n.ID)]; notice != "" {
					ui.Text(c, notice).FontSize(theme.Caption).TextColor(a.theme.Warning)
				}
			})
		})
		if place.Clicked() {
			a.jumpNote(project, r, n)
		}
		if r.Done() {
			return
		}
		if ui.Button(c, a.text("Edit", "编辑")).Height(24).Clicked() {
			a.editNote(v, r, n)
		}
		if ui.Button(c, a.text("Delete", "删除")).Height(24).Clicked() {
			a.report(a.storeReview(review.DeleteNote(r, n.ID)))
		}
	})
}

// noteLabel says where a note is, as a person or an agent reads a place.
func noteLabel(anchor review.Anchor) string {
	label := anchor.Path
	if l := anchor.Lines; l != nil && l.First == l.Last {
		label += fmt.Sprintf(":%d", l.First)
	} else if l != nil {
		label += fmt.Sprintf(":%d-%d", l.First, l.Last)
	}
	if anchor.Side == review.SideOld {
		label += " (old)"
	}
	return label
}

// noteScope says what the view showed as the note was made.
func (a *app) noteScope(anchor review.Anchor) string {
	scope := a.text("File", "文件")
	if d := anchor.Diff; d != nil {
		scope = "Diff · " + cmp.Or(d.Commit, d.Base)
		if d.Uncommitted {
			scope = "Diff · " + a.text("Uncommitted Only", "只看未提交")
		}
	}
	if anchor.Uncommitted {
		scope += " · " + a.text("uncommitted", "未提交")
	}
	return scope
}

// firstLines is the text's first n lines, with … ending it when there are more.
func firstLines(text string, n int) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		return append(lines[:n:n], "…")
	}
	return lines
}
