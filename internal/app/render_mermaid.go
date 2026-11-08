package app

import (
	"errors"
	"fmt"

	"github.com/bencode/jun/internal/theme"
	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// mermaidDiagrams builds a page's mermaid code blocks as diagrams, each laid
// out once.
type mermaidDiagrams struct {
	a    *app
	made map[diagramKey]*diagram
}

type diagramKey struct{ src, family string }

// diagram is a block laid out and ready to paint, or why it could not be.
type diagram struct {
	w, h float32
	ops  []paintOp // in the scene's order
	err  error
}

// paintOp paints a path or a line of text of a diagram.
type paintOp func(p *ui.Painter, ink inkFunc)

// inkFunc is the color a paint paints in, or false when it paints nothing.
type inkFunc func(scene.Paint) (ui.Color, bool)

// block shows a diagram no wider than the page and no larger than it was laid
// out, or the code while the block is still being written. A diagram sumi
// cannot draw stays code, with the reason below it.
func (d *mermaidDiagrams) block(c *ui.Context, b *markdown.CodeBlockView) {
	if b.Open {
		markdown.DefaultCodeBlock(c, b)
		return
	}
	dg := d.diagram(b.Code(), c.Theme().Font)
	if dg.err != nil {
		markdown.DefaultCodeBlock(c, b)
		ui.Text(c, d.a.text("Not drawn: ", "未绘制：")+dg.err.Error()).FontSize(theme.Caption).TextColor(c.Theme().TextMuted)
		return
	}
	ui.Box(c).FillWidth().MaxWidth(dg.w).AspectRatio(dg.w / dg.h).AlignSelf(ui.Center).Draw(func(p *ui.Painter, r ui.Rect) {
		ink, s := inkFor(p.Theme()), r.W/dg.w
		p.Transform(ui.Translation(r.X, r.Y), 0, 0, func() {
			p.Transform(ui.Scaling(s, s), 0, 0, func() {
				for _, op := range dg.ops {
					op(p, ink)
				}
			})
		})
	})
}

func (d *mermaidDiagrams) diagram(src, family string) *diagram {
	k := diagramKey{src, family}
	if dg := d.made[k]; dg != nil {
		return dg
	}
	if d.made == nil {
		d.made = map[diagramKey]*diagram{}
	}
	dg := layOut(src, family)
	d.made[k] = dg
	return dg
}

// layOut runs sumi on the UI thread. A panic there is reported as the block's
// error: a diagram must not take down the terminals running beside it.
func layOut(src, family string) (dg *diagram) {
	defer func() {
		if r := recover(); r != nil {
			dg = &diagram{err: fmt.Errorf("sumi failed: %v", r)}
		}
	}()
	s, err := mermaid.Layout(src, mermaid.Options{Measurer: uiMeasurer{family, map[scene.Font][2]float64{}}})
	if err != nil {
		return &diagram{err: err}
	}
	if s.Width <= 0 || s.Height <= 0 { // an empty sequence diagram: nothing to scale
		return &diagram{err: errors.New("empty diagram")}
	}
	return &diagram{w: float32(s.Width), h: float32(s.Height), ops: compile(s, family)}
}

// compile is what paints s, the items in turn, each its paths under its text.
func compile(s *scene.Scene, family string) []paintOp {
	var ops []paintOp
	for _, it := range s.Items {
		for _, sp := range it.Paths {
			ops = append(ops, pathOps(sp)...)
		}
		for _, t := range it.Texts {
			for _, l := range t.Lines {
				ops = append(ops, textOp(t, l, family))
			}
		}
	}
	return ops
}

// pathOps paints sp, a solid line a piece at a time. A Painter gives a path a
// mask of its whole bounds, so edges crossing a diagram would fill the atlas and
// leave later shapes unpainted; a piece's bounds are about as small as its ink.
// The pieces overlap in their round caps, which only an opaque line can hide,
// and dashes run along the whole path, so other strokes stay whole.
func pathOps(sp scene.Path) []paintOp {
	opaque := sp.Stroke.Paint.Token != scene.None || sp.Stroke.Paint.Color.A == 255
	if sp.Fill != (scene.Paint{}) || len(sp.Stroke.Dash) > 0 || !opaque {
		return []paintOp{pathOp(sp)}
	}
	var ops []paintOp
	for _, segs := range pieces(sp.Segs) {
		ops = append(ops, pathOp(scene.Path{Segs: segs, Stroke: sp.Stroke}))
	}
	return ops
}

// pieces are segs, one drawing command each with a move to where it starts; segs
// whole, when a Close joins them.
func pieces(segs []scene.Seg) [][]scene.Seg {
	var out [][]scene.Seg
	var at scene.Point
	for _, s := range segs {
		end := s.Pts[0]
		switch s.Op {
		case scene.Close:
			return [][]scene.Seg{segs}
		case scene.MoveTo:
			at = end
			continue
		case scene.QuadTo:
			end = s.Pts[1]
		case scene.CubeTo:
			end = s.Pts[2]
		}
		out = append(out, []scene.Seg{{Op: scene.MoveTo, Pts: [3]scene.Point{at}}, s})
		at = end
	}
	return out
}

func pathOp(sp scene.Path) paintOp {
	path := uiPath(sp.Segs)
	if len(sp.Stroke.Dash) > 0 {
		dash := make([]float32, len(sp.Stroke.Dash))
		for i, d := range sp.Stroke.Dash {
			dash[i] = float32(d)
		}
		path.Dash(0, dash...)
	}
	width := float32(sp.Stroke.Width)
	return func(p *ui.Painter, ink inkFunc) {
		if c, ok := ink(sp.Fill); ok {
			p.FillPath(path, c)
		}
		if c, ok := ink(sp.Stroke.Paint); ok && width > 0 {
			p.StrokePath(path, width, c)
		}
	}
}

func uiPath(segs []scene.Seg) *ui.Path {
	p := &ui.Path{}
	f := func(v float64) float32 { return float32(v) }
	for _, s := range segs {
		a, b, e := s.Pts[0], s.Pts[1], s.Pts[2]
		switch s.Op {
		case scene.MoveTo:
			p.MoveTo(f(a.X), f(a.Y))
		case scene.LineTo:
			p.LineTo(f(a.X), f(a.Y))
		case scene.QuadTo:
			p.QuadTo(f(a.X), f(a.Y), f(b.X), f(b.Y))
		case scene.CubeTo:
			p.CubeTo(f(a.X), f(a.Y), f(b.X), f(b.Y), f(e.X), f(e.Y))
		case scene.Close:
			p.Close()
		}
	}
	return p
}

// textOp sets a line of text on its baseline; a layout is painted from its top.
func textOp(t scene.Text, l scene.Line, family string) paintOp {
	layout := ui.ShapeText(l.S, uiFont(t.Font, family), 0)
	x, y := float32(l.X), float32(l.Baseline)-layout.Lines()[0].Baseline
	return func(p *ui.Painter, ink inkFunc) {
		if c, ok := ink(t.Paint); ok {
			p.TextLayout(layout, x, y, c)
		}
	}
}

func uiFont(f scene.Font, family string) ui.Font {
	out := ui.Font{Family: family, Size: float32(f.Size), Italic: f.Italic}
	if f.Bold {
		out.Weight = 700
	}
	if f.Mono {
		out.Family = "monospace"
	}
	return out
}

// uiMeasurer measures text as the window draws it, in the theme's font and the
// fonts it falls back to. Ascent and descent are the theme font's alone, or a
// label's height would change with its characters.
type uiMeasurer struct {
	family  string
	extents map[scene.Font][2]float64
}

func (m uiMeasurer) Measure(s string, f scene.Font) scene.Metrics {
	font := uiFont(f, m.family)
	e, ok := m.extents[f]
	if !ok {
		l := ui.ShapeText(" ", font, 0).Lines()[0]
		e = [2]float64{float64(l.Ascent), float64(l.Descent)}
		m.extents[f] = e
	}
	w, _ := ui.ShapeText(s, font, 0).Size()
	return scene.Metrics{Width: float64(w), Ascent: e[0], Descent: e[1]}
}

// inkFor settles paints in sumi's colors for th's brightness. An edge label's
// patch of background is the window's, not sumi's.
func inkFor(th *ui.Theme) inkFunc {
	pal := scene.Light
	if theme.IsDark(th) {
		pal = scene.Dark
	}
	return func(sp scene.Paint) (ui.Color, bool) {
		if sp.Token == scene.Background {
			return th.Background, true
		}
		c, ok := pal.Resolve(sp)
		return ui.Color{R: c.R, G: c.G, B: c.B, A: c.A}, ok
	}
}
