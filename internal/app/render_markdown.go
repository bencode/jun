package app

import (
	"cmp"
	"fmt"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/bencode/jun/internal/repo"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/egoist/mygo/ui/markdown/parser"
)

// markdownKind shows Markdown as a page to read: its images read from the
// same source, its links to the repository's files opening them in Jun.
var markdownKind = renderKind{
	name:  "reading",
	label: func(a *app) string { return a.text("Reading", "阅读") },
	accepts: func(name string, data []byte) bool {
		return hasExtension(name, ".md", ".markdown") && isText(data)
	},
	make: func(f fileEnv) fileRender {
		m := &markdownRender{f: f, top: f.r.top, jump: -1}
		m.options = markdown.Options{Theme: &m.theme, LoadImage: m.loadImage, OnLink: m.openLink, Block: m.noteBlock}
		return m
	},
}

type markdownRender struct {
	f       fileEnv
	top     string // the repository's, for images loading off the UI thread
	text    string
	theme   markdown.Theme // its text size; the window theme's for the rest
	options markdown.Options

	headings []heading    // as the last frame built them
	marks    []*ui.Handle // the place of each heading, by its index in headings
	scroll   ui.ScrollState
	scroller ui.Handle
	jump     int // the heading to scroll to the top in the next frame, -1 for none
}

// heading is a heading of the document, as the outline lists it.
type heading struct {
	level int
	text  string
}

// headingGap is the room left above a heading scrolled to the top.
const headingGap = 12

func (m *markdownRender) update(data []byte) { m.text = string(data) }

func (m *markdownRender) view(c *ui.Context, size float32) {
	m.theme.FontSize = size
	if m.jump >= 0 && m.jump < len(m.marks) {
		mark, top := m.marks[m.jump].Bounds(c), m.scroller.Bounds(c)
		m.scroll.ScrollTo(0, max(m.scroll.Y+mark.Y-top.Y-headingGap, 0), ui.Smooth)
	}
	m.jump = -1
	m.headings = m.headings[:0]
	ui.Scroll(c).Grow(1).FillWidth().TrackScroll(&m.scroll).Bind(&m.scroller).Children(func() {
		ui.Column(c).FillWidth().Padding(28, 32).AlignItems(ui.Center).Children(func() {
			page := markdown.View(c, m.text, &m.options).FillWidth()
			if w := markdownMaxWidths[m.f.a.markdownWidth()]; w > 0 {
				page.MaxWidth(w)
			}
		})
	})
}

// noteBlock notes each heading as MyGo builds it, with an empty mark at
// its top whose place a jump scrolls to; the heading builds as it would.
// With Block set, MyGo builds each block in a column of its own, which
// keeps a thematic break's own margins where they were overridden: the
// break is drawn here as it is without the column.
func (m *markdownRender) noteBlock(c *ui.Context, d *parser.Document, i int) bool {
	b := d.Block(i)
	if b.Kind == parser.ThematicBreak {
		th := markdown.DefaultTheme(c.Theme())
		rule := ui.Box(c).Height(1).Background(th.Rule).AlignSelf(ui.Stretch)
		if i == b.Parent+1 { // the first of its parent's blocks keeps its margins
			rule.Margin(cmp.Or(m.theme.FontSize, th.FontSize)*0.25, 0)
		}
		return true
	}
	if b.Kind != parser.Heading {
		return false
	}
	k := len(m.headings)
	if k == len(m.marks) {
		m.marks = append(m.marks, new(ui.Handle))
	}
	ui.Box(c).Height(0).Bind(m.marks[k])
	m.headings = append(m.headings, heading{level: b.Level, text: parser.PlainText(d.Inlines(i))})
	return false
}

// goTo scrolls the heading at index k of headings to the top.
func (m *markdownRender) goTo(k int) {
	m.jump = k
	if w := m.f.a.window; w != nil {
		w.Invalidate()
	}
}

func (m *markdownRender) detail() string { return "" }

// loadImage reads an image the document refers to from the file's source,
// on a goroutine of its own.
func (m *markdownRender) loadImage(dest string) (ui.ImageSource, error) {
	name, ok := repoPath(m.f.ref.Path, dest)
	if !ok {
		return nil, os.ErrNotExist
	}
	data, size, err := repo.Read(m.top, m.f.s, name, fileLimit)
	if err != nil {
		return nil, err
	}
	if size > fileLimit {
		return nil, fmt.Errorf("%s is over 10 MB", name)
	}
	return ui.DecodeImage(data)
}

// openLink opens web and mail addresses with their apps, and a file of the
// source in Jun, as a click in Files does. Other links do nothing, nor
// do anchors within the page: with OnLink set, MyGo leaves them to it.
func (m *markdownRender) openLink(dest string) {
	if ui.SafeToOpen(dest) {
		if err := mygo.Shell.OpenExternal(dest); err != nil {
			m.f.a.report(err)
		}
		return
	}
	name, ok := repoPath(m.f.ref.Path, dest)
	if !ok {
		return
	}
	// A directory, or a file not there, is not opened; before the source's
	// files are read, the file view says what is wrong.
	if l := m.f.a.listingFor(m.f.r, m.f.s); l.files != nil && !slices.Contains(l.files, name) {
		return
	}
	m.f.a.openFile(m.f.project, fileNode{source: m.f.s.Key(), path: name}, false)
}

// repoPath resolves a link's destination to a path in the repository: in
// the file's directory, or with a leading slash in the repository's top
// level. Addresses (https:, data:, …) and paths outside it resolve to
// nothing.
func repoPath(file, dest string) (string, bool) {
	dest, _, _ = strings.Cut(dest, "#")
	dest, _, _ = strings.Cut(dest, "?")
	dest, err := url.PathUnescape(dest)
	if err != nil || dest == "" || strings.Contains(dest, ":") || strings.HasPrefix(dest, "//") {
		return "", false
	}
	name := path.Join(path.Dir(file), dest)
	if strings.HasPrefix(dest, "/") {
		name = path.Clean(strings.TrimPrefix(dest, "/"))
	}
	return name, repo.Valid(name)
}
