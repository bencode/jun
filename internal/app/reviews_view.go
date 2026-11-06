package app

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/review"
	"github.com/bencode/jun/internal/theme"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// reviewsViewState is a Reviews view's choice.
type reviewsViewState struct {
	list     ui.ListState
	selected int
}

// reviewRow is a row of the list: a day's heading, or a review.
type reviewRow struct {
	day    string
	review review.Review
}

// reviewRows heads each day's reviews with the day, as Sessions does.
func reviewRows(reviews []review.Review, now time.Time, text func(en, zh string) string) []reviewRow {
	var rows []reviewRow
	last := ""
	for _, r := range reviews {
		if day := dayName(r.CreatedAt.Local(), now.Local(), text); day != last {
			rows = append(rows, reviewRow{day: day})
			last = day
		}
		rows = append(rows, reviewRow{review: r})
	}
	return rows
}

// reviewsView lists the project's reviews: only what each was, as Sessions
// lists sessions; a click shows one in a preview tab, a double click or
// Enter in a tab of its own.
func (a *app) reviewsView(c *ui.Context, v *layout.View) {
	project := a.project(v.ProjectName)
	if project == nil {
		return
	}
	l := a.reviewsFor(project)
	st := a.reviewViews[v.ID]
	if st == nil {
		st = &reviewsViewState{selected: -1}
		st.list.Selected = &st.selected
		a.reviewViews[v.ID] = st
	}
	ui.Column(c.Key(v.ID)).Fill().Gap(0).Children(func() {
		if l.err != nil {
			ui.Text(c, l.err.Error()).FontSize(theme.Caption).TextColor(a.theme.Danger).Padding(4, 12)
		}
		rows := reviewRows(l.items, time.Now(), a.text)
		if len(rows) == 0 {
			ui.Text(c, a.text("No reviews in this project yet.", "这个项目还没有 Review。")).TextColor(a.theme.TextMuted).Padding(12)
			return
		}
		a.reviewList(c, st, rows, project)
	})
}

func (a *app) reviewList(c *ui.Context, st *reviewsViewState, rows []reviewRow, project *layout.Project) {
	st.list.Key = func(i int) any {
		if rows[i].day != "" {
			return "day:" + rows[i].day
		}
		return rows[i].review.ID
	}
	st.list.Header = func(i int) bool { return rows[i].day != "" }
	st.list.Label = func(i int) string {
		if rows[i].day != "" {
			return rows[i].day
		}
		return diffSourceName(rows[i].review.Source)
	}
	list := ui.List(c, &st.list, len(rows), func(i int) {
		row := rows[i]
		if row.day != "" {
			ui.Text(c, row.day).FontSize(theme.Caption).Bold().TextColor(a.theme.TextMuted).
				FillWidth().Padding(8, 8, 4).Background(a.theme.Background)
			return
		}
		a.reviewRow(c, row.review, project)
	}).Grow(1).FillWidth()
	chosen := func() (review.Review, bool) {
		if st.selected < 0 || st.selected >= len(rows) || rows[st.selected].day != "" {
			return review.Review{}, false
		}
		return rows[st.selected].review, true
	}
	if list.Changed() {
		if r, ok := chosen(); ok {
			a.openReview(project.Name, r.ID, false)
		}
	}
	if list.Submitted() {
		if r, ok := chosen(); ok {
			a.openReview(project.Name, r.ID, true)
		}
	}
}

func (a *app) reviewRow(c *ui.Context, r review.Review, project *layout.Project) {
	ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Start).Padding(4, 8).ContextMenu(func(m *ui.Menu) {
		if m.Item(a.text("Open", "打开")).Chosen() {
			a.openReview(project.Name, r.ID, true)
		}
		if m.Item(a.text("Copy Address", "复制地址")).Chosen() {
			a.copyReviewAddress(r)
		}
		if m.Item(a.text("Delete…", "删除…")).Chosen() {
			a.confirmDeleteReview(r)
		}
	}).Children(func() {
		mark, color := "●", a.theme.Accent
		if r.Done() {
			mark, color = "✓", a.theme.Success
		}
		ui.Text(c, mark).FontSize(theme.Caption).TextColor(color).Shrink(0).Width(14)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, diffSourceName(r.Source)).MinWidth(0).SingleLine().Ellipsis("…")
			ui.Text(c, a.reviewDetail(r)).FontSize(theme.Caption).TextColor(a.theme.TextMuted).MinWidth(0).SingleLine().Ellipsis("…")
		})
	})
}

// reviewDetail is a row's second line: what the review was of, and when.
func (a *app) reviewDetail(r review.Review) string {
	uncommitted := ""
	if r.Uncommitted() {
		uncommitted = a.text("uncommitted", "未提交")
	}
	parts := []string{r.Head, uncommitted, r.CreatedAt.Local().Format("15:04"), a.noteCountName(len(r.Notes))}
	return strings.Join(slices.DeleteFunc(parts, func(p string) bool { return p == "" }), " · ")
}

func (a *app) noteCountName(n int) string {
	if n == 1 {
		return a.text("1 note", "1 条")
	}
	return fmt.Sprintf(a.text("%d notes", "%d 条"), n)
}

// openReview shows the review where new tabs go: in the preview tab, which
// the next one opened replaces, unless pinned.
func (a *app) openReview(project, id string, pinned bool) {
	target := a.tabsTarget()
	if target == nil {
		return
	}
	shown := slices.ContainsFunc(target.Views, func(v *layout.View) bool { return v.Review == id && v.ProjectName == project })
	preview := a.previewTabs[target.ID]
	view, err := a.state.OpenReview(target.ID, project, id, preview)
	if err != nil {
		a.report(err)
		return
	}
	if pinned && view == preview {
		delete(a.previewTabs, target.ID)
	} else if !pinned && !shown {
		a.previewTabs[target.ID] = view
	}
	a.layoutChanged()
}

// showReviews shows the side bar's Reviews.
func (a *app) showReviews() {
	p := a.state.CurrentPerspective()
	if p == nil {
		return
	}
	i := slices.IndexFunc(p.Side.Views, func(v *layout.View) bool { return v.Kind == layout.ViewReviews })
	if i < 0 {
		return
	}
	a.report(a.state.RevealSideView(p.Side.Views[i].ID))
	a.layoutChanged()
}

func (a *app) toggleSideView(id layout.ViewID) {
	a.report(a.state.ToggleSideView(id))
	a.layoutChanged()
}

func (a *app) copyReviewAddress(r review.Review) {
	mygo.Clipboard.WriteText(review.Path(a.reviewRoot, r))
}

func (a *app) confirmDeleteReview(r review.Review) {
	a.confirm(fmt.Sprintf(a.text("Delete the review of %s?", "删除 %s 的 Review？"), diffSourceName(r.Source)),
		a.text("Its notes will be gone.", "其中的标注也会一起删除。"), a.text("Delete", "删除"), func() { a.deleteReview(r) })
}
