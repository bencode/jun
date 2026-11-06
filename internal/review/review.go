// Package review keeps what the user notes while reading a branch's changes
// and files: one record a review, with a file each, for an agent to read.
package review

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"
	"time"

	"github.com/bencode/jun/internal/layout"
)

// Side is the side of a comparison a note is on: files have only the new.
type Side string

const (
	SideNew Side = "new"
	SideOld Side = "old"
)

// Lines are lines First to Last, from 1, inclusive.
type Lines struct {
	First int `json:"first"`
	Last  int `json:"last"`
}

// Anchor is where a note was made: what the view showed, for a jump to
// show it again, and the text there, which finds the place once lines moved.
type Anchor struct {
	Diff        *layout.DiffRef `json:"diff,omitempty"`   // what the Diff compared; nil for a file tab
	Render      string          `json:"render,omitempty"` // how a file tab showed the file
	Path        string          `json:"path"`             // as FileRef.Path has it
	Side        Side            `json:"side"`
	Lines       *Lines          `json:"lines,omitempty"` // nil for the whole file
	Quote       string          `json:"quote,omitempty"`
	Match       string          `json:"match,omitempty"` // the source of a quote read from a rendered page
	Head        string          `json:"head,omitempty"`  // the source's commit then
	Uncommitted bool            `json:"uncommitted,omitempty"`
}

// Note is what the user said of a place.
type Note struct {
	ID     int    `json:"id"`
	Anchor Anchor `json:"anchor"`
	Text   string `json:"text"`
}

// Review is one reading of a source of a project: its notes, open until
// the user is done with it, then read-only.
type Review struct {
	ID        string     `json:"id"`
	Project   string     `json:"project"` // the project's directory, which a rename keeps
	Source    string     `json:"source"`  // as FileRef.Source has it
	Head      string     `json:"head,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	DoneAt    *time.Time `json:"doneAt,omitempty"`
	Notes     []Note     `json:"notes"`
}

// New starts a review of the source at the commit head.
func New(project, source, head string, now time.Time) Review {
	suffix := make([]byte, 2)
	rand.Read(suffix) // never fails, as crypto/rand documents
	id := now.Format("20060102-150405") + "-" + hex.EncodeToString(suffix)
	return Review{ID: id, Project: project, Source: source, Head: head, CreatedAt: now, Notes: []Note{}}
}

// Done reports whether the user finished the review.
func (r Review) Done() bool { return r.DoneAt != nil }

// Uncommitted reports whether any note is on text not committed.
func (r Review) Uncommitted() bool {
	return slices.ContainsFunc(r.Notes, func(n Note) bool { return n.Anchor.Uncommitted })
}

// AddNote is the review with a new note.
func AddNote(r Review, a Anchor, text string) Review {
	id := 0
	for _, n := range r.Notes {
		id = max(id, n.ID)
	}
	r.Notes = append(slices.Clone(r.Notes), Note{ID: id + 1, Anchor: a, Text: text})
	return r
}

// EditNote is the review with the note saying text instead.
func EditNote(r Review, id int, text string) Review {
	r.Notes = slices.Clone(r.Notes)
	if i := slices.IndexFunc(r.Notes, func(n Note) bool { return n.ID == id }); i >= 0 {
		r.Notes[i].Text = text
	}
	return r
}

// DeleteNote is the review without the note.
func DeleteNote(r Review, id int) Review {
	r.Notes = slices.DeleteFunc(slices.Clone(r.Notes), func(n Note) bool { return n.ID == id })
	return r
}

// Finish is the review done, which takes a note at least.
func Finish(r Review, now time.Time) (Review, error) {
	if len(r.Notes) == 0 {
		return r, errors.New("a review needs a note")
	}
	r.DoneAt = &now
	return r, nil
}
