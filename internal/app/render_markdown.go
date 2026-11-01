package app

import (
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
)

// markdownKind shows Markdown as a page to read: its images read from the
// same source, its links to the repository's files opening them in Jun.
var markdownKind = renderKind{
	label: func(a *app) string { return a.text("Reading", "阅读") },
	accepts: func(name string, data []byte) bool {
		return hasExtension(name, ".md", ".markdown") && isText(data)
	},
	make: func(f fileEnv) fileRender {
		m := &markdownRender{f: f, top: f.r.top}
		m.options = markdown.Options{Theme: &m.theme, LoadImage: m.loadImage, OnLink: m.openLink}
		return m
	},
}

type markdownRender struct {
	f       fileEnv
	top     string // the repository's, for images loading off the UI thread
	text    string
	theme   markdown.Theme // its text size; the window theme's for the rest
	options markdown.Options
}

func (m *markdownRender) update(data []byte) { m.text = string(data) }

func (m *markdownRender) view(c *ui.Context, size float32) {
	m.theme.FontSize = size
	ui.Scroll(c).Grow(1).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(28, 32).AlignItems(ui.Center).Children(func() {
			markdown.View(c, m.text, &m.options).FillWidth().MaxWidth(760)
		})
	})
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
	m.f.a.openFile(m.f.project, fileNode{m.f.s.Key(), name}, false)
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
