package app

import (
	"slices"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/review"
)

// reviewList is a project's reviews, the latest first: read once, then
// kept here as Jun alone writes them.
type reviewList struct {
	items []review.Review
	err   error
	read  bool
}

// reviewsFor returns the project's reviews, reading them the first time.
func (a *app) reviewsFor(project *layout.Project) *reviewList {
	l := a.reviewLists[project.CWD]
	if l == nil {
		l = &reviewList{}
		a.reviewLists[project.CWD] = l
		root, dir := a.reviewRoot, project.CWD
		var items []review.Review
		var err error
		a.background(func() { items, err = review.List(root, dir) }, func() {
			l.items, l.err, l.read = items, err, true
			a.reviewsChanged++
		})
	}
	return l
}

// find is the review with the id.
func (l *reviewList) find(id string) (review.Review, bool) {
	i := slices.IndexFunc(l.items, func(r review.Review) bool { return r.ID == id })
	if i < 0 {
		return review.Review{}, false
	}
	return l.items[i], true
}

// open is the source's review the user has not finished: at most one, as
// saveNote starts one only without it.
func (l *reviewList) open(source string) (review.Review, bool) {
	i := slices.IndexFunc(l.items, func(r review.Review) bool { return r.Source == source && !r.Done() })
	if i < 0 {
		return review.Review{}, false
	}
	return l.items[i], true
}

// storeReview writes the review, and takes it only once it is written.
func (a *app) storeReview(r review.Review) error {
	if err := review.Save(a.reviewRoot, r); err != nil {
		return err
	}
	l := a.reviewLists[r.Project]
	if i := slices.IndexFunc(l.items, func(x review.Review) bool { return x.ID == r.ID }); i >= 0 {
		l.items[i] = r
	} else {
		l.items = append([]review.Review{r}, l.items...)
	}
	a.reviewsChanged++
	return nil
}

// deleteReview removes the review and its tabs here; a tab of it in
// another perspective says it is gone.
func (a *app) deleteReview(r review.Review) {
	if err := review.Delete(a.reviewRoot, r); err != nil {
		a.report(err)
		return
	}
	l := a.reviewLists[r.Project]
	l.items = slices.DeleteFunc(slices.Clone(l.items), func(x review.Review) bool { return x.ID == r.ID })
	a.reviewsChanged++
	a.report(a.closeTabs(func(v *layout.View) bool { return v.Review == r.ID }, nil))
}
