package app

import (
	"path"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// sourceKind shows text as it is, read-only, highlighted by the file's
// name.
var sourceKind = renderKind{
	label:   func(a *app) string { return a.text("Source", "源码") },
	accepts: func(_ string, data []byte) bool { return isText(data) },
	make:    func(f fileEnv) fileRender { return &sourceRender{name: f.ref.Path} },
}

type sourceRender struct {
	name   string
	editor *editor.State
}

// update keeps the editor, and its place, as the text changes.
func (s *sourceRender) update(data []byte) {
	text := string(data)
	if s.editor == nil {
		buffer := editor.NewBuffer(text)
		buffer.SetLanguage(editor.LanguageFor(path.Base(s.name)))
		s.editor = &editor.State{Buffer: buffer, ReadOnly: true, Label: s.name}
	} else if s.editor.Buffer.String() != text {
		s.editor.Buffer.Reset(text)
	}
}

func (s *sourceRender) view(c *ui.Context) {
	if s.editor != nil {
		editor.View(c, s.editor).Grow(1).FillWidth()
	}
}

func (s *sourceRender) detail() string { return "" }
