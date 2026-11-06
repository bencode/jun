package app

import (
	"bytes"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/bencode/jun/internal/layout"
	"github.com/bencode/jun/internal/repo"
	"github.com/bencode/jun/internal/review"
	"github.com/egoist/mygo/ui"
)

// fileRender is one way a file view shows a file: its state, kept as the
// file is read again so that its place (scroll, selection) stays.
type fileRender interface {
	update(data []byte)               // the content, read (again); on the UI thread
	view(c *ui.Context, size float32) // size: the text size in DIPs
	detail() string                   // what the header adds, as an image's size; "" for nothing
	spot() noteSpot                   // what is selected, to note
	showMarks(marks []noteMark)       // the notes on the file, to show
	// goToPlace goes to the place, when it has lines, and reports whether it
	// could yet: not before it laid out.
	goToPlace(place review.Place, lines bool) bool
}

// renderKind is a way of showing files: which files it can show, and how
// it starts showing one.
type renderKind struct {
	name    string              // as a note's anchor keeps it
	label   func(a *app) string // its name in the header's switch
	accepts func(name string, data []byte) bool
	make    func(f fileEnv) fileRender
}

// fileEnv is what a render knows of its file: the app, the repository,
// the source the file is read from, and the file.
type fileEnv struct {
	a       *app
	r       *repoState
	s       repo.Source
	ref     layout.FileRef
	project string
	view    *layout.View
}

// renderKinds in order of preference: a file shows with the first that
// accepts it, and the header's switch offers the others that do.
var renderKinds = []renderKind{markdownKind, imageKind, sourceKind}

// kindsFor lists the kinds that can show the file, by their index in
// renderKinds.
func kindsFor(name string, data []byte) []int {
	var kinds []int
	for i, k := range renderKinds {
		if k.accepts(name, data) {
			kinds = append(kinds, i)
		}
	}
	return kinds
}

// isText reports whether data reads as text: UTF-8, without NUL bytes.
func isText(data []byte) bool {
	return bytes.IndexByte(data, 0) < 0 && utf8.Valid(data)
}

// hasExtension reports whether the file's name ends in one of exts,
// ignoring case.
func hasExtension(name string, exts ...string) bool {
	return slices.Contains(exts, strings.ToLower(path.Ext(name)))
}
