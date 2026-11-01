package app

import (
	"bytes"
	"io"
	"log"
	"sync"
	"time"

	"github.com/bencode/jun/internal/terminal"
	"github.com/bencode/jun/internal/theme"
	ghostty "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// discardConn connects a preview to nothing: typing into it goes nowhere,
// and it shows only what Jun feeds it. Its view tells it the grid it fits.
type discardConn struct {
	done  chan struct{}
	once  sync.Once
	mu    sync.Mutex
	fit   [2]int // the view's columns and rows
	onFit func(rows int)
}

func (c *discardConn) Read([]byte) (int, error)    { <-c.done; return 0, io.EOF }
func (c *discardConn) Write(b []byte) (int, error) { return len(b), nil }
func (c *discardConn) Close() error                { c.once.Do(func() { close(c.done) }); return nil }

func (c *discardConn) Resize(cols, rows int) error {
	c.mu.Lock()
	c.fit = [2]int{cols, rows}
	c.mu.Unlock()
	if c.onFit != nil {
		c.onFit(rows)
	}
	return nil
}

func (c *discardConn) fitted() [2]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fit
}

// forgetColumns drops the columns the view fit at a font no longer used.
func (c *discardConn) forgetColumns() {
	c.mu.Lock()
	c.fit[0] = 0
	c.mu.Unlock()
}

// A preview is a read-only copy of an agent's terminal: a terminal shows
// in one view at a time, so the card shows a copy fed its snapshots. The
// copy has the source's columns and the rows its view fits: a snapshot
// replays the screen line by line, so a copy shorter than the source
// shows the bottom of what the screen holds, where agents ask and work.
type preview struct {
	term *ghostty.Terminal
	conn *discardConn

	mu       sync.Mutex
	snapshot []byte // as last fed
	fed      [2]int // the size it was fed at

	fontSize float32   // the UI thread's
	width    float32   // the UI thread's: the area the font was guessed for
	cols     int       // the UI thread's: the columns it was guessed for
	seen     time.Time // the UI thread's: when its card last showed
}

// newPreview starts an empty copy, which the next poll fills: a snapshot
// and its feed take too long for the UI thread as cards scroll in. The copy
// keeps no scrollback, as a card shows only the screen; invalidate asks
// for a frame once its view fits a new grid.
func newPreview(invalidate func()) (*preview, error) {
	p := &preview{conn: &discardConn{done: make(chan struct{})}, seen: time.Now()}
	term, err := ghostty.New(ghostty.Options{Conn: p.conn, ResizeByConn: true, Theme: theme.JunTerminal(), Scrollback: -1})
	if err != nil {
		return nil, err
	}
	p.term = term
	p.conn.onFit = func(rows int) {
		p.mu.Lock()
		if p.snapshot != nil {
			p.show(p.snapshot, p.fed[0], rows)
		}
		p.mu.Unlock()
		invalidate()
	}
	return p, nil
}

// syncUpdate is in a snapshot taken while a program holds a synchronized
// update (mode 2026), as Claude Code does as it redraws: the screen is half
// drawn, and the preview fed it would hold its frames.
var syncUpdate = []byte("\x1b[?2026h")

const syncRetry = 20 * time.Millisecond

// refresh shows the source's screen when it changed, trying a few times
// for one no synchronized update holds; failing that, the last one stays.
func (p *preview) refresh(source *ghostty.Terminal, attempts int) {
	snapshot := source.Snapshot()
	for i := 1; i < attempts && bytes.Contains(snapshot, syncUpdate); i++ {
		time.Sleep(syncRetry)
		snapshot = source.Snapshot()
	}
	if bytes.Contains(snapshot, syncUpdate) {
		return
	}
	cols, rows := source.Size()
	if fit := p.conn.fitted(); fit[1] > 0 {
		rows = fit[1]
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.show(snapshot, cols, rows)
}

// show feeds the snapshot at a size, unless it shows it already; p.mu is
// held.
func (p *preview) show(snapshot []byte, cols, rows int) {
	if bytes.Equal(snapshot, p.snapshot) && p.fed == [2]int{cols, rows} {
		return
	}
	p.snapshot, p.fed = snapshot, [2]int{cols, rows}
	if c, r := p.term.Size(); [2]int{c, r} != p.fed {
		p.term.Resize(cols, rows)
	}
	// One write: a frame never shows the cleared screen.
	p.term.Feed(append([]byte("\x1bc\x1b[3J"), snapshot...))
}

// fit sizes the preview's font, from 6 to 13, for its columns to fill
// area's width: first a guess, a cell of the system's monospace font being
// about 0.6em wide besides the view's padding of 6 a side, then smaller as
// long as the view says it fits fewer. It reports whether the size changed.
func (p *preview) fit(area ui.Rect) bool {
	if area.W <= 0 || area.H <= 0 {
		return false
	}
	cols, _ := p.term.Size()
	size := p.fontSize
	if area.W != p.width || cols != p.cols {
		p.width, p.cols = area.W, cols
		size = (area.W - 12) / (float32(cols) * 0.6)
	} else if fit := p.conn.fitted(); fit[0] > 0 && fit[0] < cols {
		size = min(size*float32(fit[0])/float32(cols), size-0.25)
	}
	size = min(max(size, 6), 13)
	if size == p.fontSize {
		return false
	}
	p.fontSize = size
	p.conn.forgetColumns()
	p.term.SetFont(ghostty.Font{Size: size})
	return true
}

func (a *app) previewFor(session *terminal.Session) *preview {
	if p := a.previews[session.ID]; p != nil {
		p.seen = time.Now()
		return p
	}
	p, err := newPreview(func() {
		if a.window != nil {
			a.window.Invalidate()
		}
	})
	if err != nil {
		log.Printf("Jun: preview terminal %d: %v", session.ID, err)
		return nil
	}
	p.term.SetTheme(a.terminals.Theme(), nil)
	a.previews[session.ID] = p
	return p
}

const previewIdle = 30 * time.Second

// closeStalePreviews frees previews of agents gone or out of view a while.
func (a *app) closeStalePreviews() {
	for id, p := range a.previews {
		if _, ok := a.agents[id]; ok && time.Since(p.seen) < previewIdle {
			continue
		}
		a.closePreview(id)
	}
}

func (a *app) closePreview(id terminal.ID) {
	if err := a.previews[id].term.Close(); err != nil {
		log.Printf("Jun: close preview %d: %v", id, err)
	}
	delete(a.previews, id)
}

func (a *app) closePreviews() {
	for id := range a.previews {
		a.closePreview(id)
	}
}
