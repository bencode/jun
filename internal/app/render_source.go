package app

import (
	"path"
	"strings"

	"github.com/bencode/jun/internal/review"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// sourceKind shows text as it is, read-only, highlighted by the file's
// name.
var sourceKind = renderKind{
	name:    "source",
	label:   func(a *app) string { return a.text("Source", "源码") },
	accepts: func(_ string, data []byte) bool { return isText(data) },
	make:    func(f fileEnv) fileRender { return &sourceRender{f: f} },
}

type sourceRender struct {
	f      fileEnv
	editor *editor.State
	noted  []noteMark
}

// update keeps the editor, and its place, as the text changes.
func (s *sourceRender) update(data []byte) {
	text := string(data)
	if s.editor == nil {
		buffer := editor.NewBuffer(text)
		buffer.SetLanguage(editor.LanguageFor(path.Base(s.f.ref.Path)))
		s.editor = &editor.State{Buffer: buffer, ReadOnly: true, Label: s.f.ref.Path, OnGutterClick: s.gutterClick, ContextMenu: s.contextMenu}
	} else if s.editor.Buffer.String() != text {
		s.editor.Buffer.Reset(text)
	}
}

func (s *sourceRender) view(c *ui.Context, size float32) {
	if s.editor != nil {
		s.editor.Font.Size = size // in the system's monospaced font, as Family is empty
		editor.View(c, s.editor).Grow(1).FillWidth()
	}
}

func (s *sourceRender) detail() string { return "" }

// spot is the lines the selection touches, else the caret's.
func (s *sourceRender) spot() noteSpot {
	if s.editor == nil {
		return noteSpot{}
	}
	sel, buffer := s.editor.Selection(), s.editor.Buffer
	first, end := buffer.Pos(sel.Start()), buffer.Pos(sel.End())
	last := end.Line
	if end.Col == 0 && last > first.Line {
		last-- // a selection of whole lines ends on the next
	}
	quote := make([]string, 0, last-first.Line+1)
	for i := first.Line; i <= last; i++ {
		quote = append(quote, buffer.Line(i))
	}
	return noteSpot{lines: &review.Lines{First: first.Line + 1, Last: last + 1}, quote: strings.Join(quote, "\n")}
}

func (s *sourceRender) showMarks(marks []noteMark) {
	s.noted = marks
	if s.editor == nil {
		return
	}
	s.editor.GutterMarks, s.editor.LineBackgrounds = nil, nil
	for _, m := range marks {
		color := s.f.a.theme.Accent
		s.editor.GutterMarks = append(s.editor.GutterMarks, editor.GutterMark{Line: m.lines.First - 1, Color: color})
		s.editor.LineBackgrounds = append(s.editor.LineBackgrounds,
			editor.LineBackground{First: m.lines.First - 1, Last: m.lines.Last - 1, Color: color.Alpha(0.12)})
	}
}

func (s *sourceRender) goToPlace(place review.Place, lines bool) bool {
	if s.editor == nil {
		return false
	}
	s.editor.Selection() // readies the editor, for its height to be asked
	if s.editor.ViewportHeight() == 0 {
		return false
	}
	if !lines {
		return true
	}
	buffer := s.editor.Buffer
	first, last := place.Lines.First-1, place.Lines.Last-1
	s.editor.GoTo(editor.Pos{Line: first})
	s.editor.Select(buffer.Offset(editor.Pos{Line: first}), buffer.Offset(editor.Pos{Line: last, Col: len(buffer.Line(last))}))
	return true
}

// gutterClick follows a noted line to its review, or starts a note on the
// line.
func (s *sourceRender) gutterClick(line int) {
	a := s.f.a
	for _, m := range s.noted {
		if m.lines.First-1 <= line && line <= m.lines.Last-1 {
			if project := a.project(s.f.project); project != nil {
				if r, ok := a.reviewsFor(project).open(s.f.ref.Source); ok {
					a.openReview(s.f.project, r.ID, false)
				}
			}
			return
		}
	}
	buffer := s.editor.Buffer
	s.editor.Select(buffer.Offset(editor.Pos{Line: line}), buffer.Offset(editor.Pos{Line: line, Col: len(buffer.Line(line))}))
	a.noteInFile(s.f.view)
}

func (s *sourceRender) contextMenu(m *ui.Menu) {
	a := s.f.a
	if m.Item(a.text("Add Note", "添加标注")).Shortcut(noteKeys.mods, noteKeys.key).Chosen() {
		a.noteInFile(s.f.view)
	}
}
