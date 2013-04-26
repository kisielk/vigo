package editor

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/nsf/termbox-go"
	"github.com/nsf/tulib"
	"os"
	"path/filepath"
	"strconv"
)

// this is a structure which represents a key press, used for keyboard macros
type keyEvent struct {
	mod termbox.Modifier
	_   [1]byte
	key termbox.Key
	ch  rune
}

func createKeyEvent(ev *termbox.Event) keyEvent {
	return keyEvent{
		mod: ev.Mod,
		key: ev.Key,
		ch:  ev.Ch,
	}
}

func (k keyEvent) toTermboxEvent() termbox.Event {
	return termbox.Event{
		Type: termbox.EventKey,
		Mod:  k.mod,
		Key:  k.key,
		Ch:   k.ch,
	}
}

var ErrQuit = errors.New("quit")

type editor struct {
	uiBuf        tulib.Buffer
	active       *viewTree // this one is always a leaf node
	views        *viewTree // a root node
	buffers      []*buffer
	lastCmdClass vCommandClass
	statusBuf    bytes.Buffer
	quitFlag     bool
	Events       chan termbox.Event
	killBuffer_  []byte

	cutBuffers *cutBuffers

	mode    editorMode
	overlay Overlay
}

func (g *editor) quit() {
	g.setStatus("Quit")
	// Signals event loop to quit on next iteration.
	g.quitFlag = true
}

func NewEditor(filenames []string) *editor {
	g := new(editor)
	g.buffers = make([]*buffer, 0, 20)
	g.cutBuffers = newCutBuffers()

	for _, filename := range filenames {
		//TODO: Check errors here
		g.newBufferFromFile(filename)
	}
	if len(g.buffers) == 0 {
		buf := newEmptyBuffer()
		buf.name = g.bufferName("unnamed")
		g.buffers = append(g.buffers, buf)
	}
	g.views = newViewTreeLeaf(nil, newView(g.viewContext(), g.buffers[0]))
	g.active = g.views
	g.setMode(newNormalMode(g))
	g.Events = make(chan termbox.Event, 20)
	return g
}

type cutBuffers map[byte][]byte

func newCutBuffers() *cutBuffers {
	c := make(cutBuffers, 36)
	return &c
}

// UpdateAnon the contents of the anonymous cut buffer 1
// with the given byte slice s, and rotates the rest of the buffers
func (bs *cutBuffers) updateAnon(s []byte) {
	bufs := *bs
	for i := byte('9'); i > '1'; i-- {
		bufs[i] = bufs[i-1]
	}
	bufs['1'] = s
	*bs = bufs
}

// validCutBuffer panics if b is not a valid cut buffer name
// b must a character between a-z, 1-9, or .
func validCutBuffer(b byte) {
	if b != '.' && b < '1' || b > '9' && b < 'a' || b > 'z' {
		panic(fmt.Errorf("invalid cut buffer: %q", b))
	}
}

// Set updates the contents of the cut buffer b with the byte slice s
func (bs *cutBuffers) set(b byte, s []byte) {
	validCutBuffer(b)
	(*bs)[b] = s
}

// Append appends the byte slice s to the contents of buffer b
func (bs *cutBuffers) append(b byte, s []byte) {
	validCutBuffer(b)
	(*bs)[b] = append((*bs)[b], s...)
}

// Get returns the contents of the cut buffer b
func (bs *cutBuffers) get(b byte) []byte {
	validCutBuffer(b)
	return (*bs)[b]
}

func (g *editor) killBuffer(buf *buffer) {
	var replacement *buffer
	views := make([]*view, len(buf.views))
	copy(views, buf.views)

	// find replacement buffer
	if len(views) > 0 {
		for _, gbuf := range g.buffers {
			if gbuf == buf {
				continue
			}
			replacement = gbuf
			break
		}
		if replacement == nil {
			replacement = newEmptyBuffer()
			replacement.name = g.bufferName("unnamed")
			g.buffers = append(g.buffers, replacement)
		}
	}

	// replace the buffer we're killing with replacement one for
	// all the views
	for _, v := range views {
		v.attach(replacement)
	}

	// remove buffer from the list
	bi := -1
	for i, n := 0, len(g.buffers); i < n; i++ {
		if g.buffers[i] == buf {
			bi = i
			break
		}
	}

	if bi == -1 {
		panic("removing non-existent buffer")
	}

	copy(g.buffers[bi:], g.buffers[bi+1:])
	g.buffers = g.buffers[:len(g.buffers)-1]
}

func (g *editor) findBufferByFullPath(path string) *buffer {
	for _, buf := range g.buffers {
		if buf.path == path {
			return buf
		}
	}
	return nil
}

// GetBuffer returns a buffer by name, or nil if there is no such buffer
func (g *editor) getBuffer(name string) *buffer {
	for _, buf := range g.buffers {
		if buf.name == name {
			return buf
		}
	}
	return nil
}

// BufferName generates a buffer name based on the one given.
func (e *editor) bufferName(name string) string {
	if buf := e.getBuffer(name); buf == nil {
		return name
	}

	for i := 2; i < 9999; i++ {
		candidate := name + " <" + strconv.Itoa(i) + ">"
		if buf := e.getBuffer(candidate); buf == nil {
			return candidate
		}
	}
	panic("too many buffers opened with the same name")
}

func (g *editor) newBufferFromFile(filename string) (*buffer, error) {
	fullpath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("couldn't determine absolute path: %s", err)
	}
	buf := g.findBufferByFullPath(fullpath)
	if buf != nil {
		return buf, nil
	}

	f, err := os.Open(fullpath)
	if err == os.ErrNotExist {
		// Assume a new file
		g.setStatus("(New file)")
		buf = newEmptyBuffer()
	} else if err != nil {
		g.setStatus(err.Error())
		return nil, err
	}
	defer f.Close()
	buf, err = newBuffer(f)
	if err != nil {
		g.setStatus(err.Error())
		return nil, err
	}
	buf.path = fullpath

	buf.name = g.bufferName(filename)
	g.buffers = append(g.buffers, buf)
	return buf, nil
}

func (g *editor) setStatus(format string, args ...interface{}) {
	g.statusBuf.Reset()
	fmt.Fprintf(&g.statusBuf, format, args...)
}

func (g *editor) splitHorizontally() {
	if g.active.Width == 0 {
		return
	}
	g.active.splitHorizontally()
	g.active = g.active.left
	g.Resize()
}

func (g *editor) splitVertically() {
	if g.active.Height == 0 {
		return
	}
	g.active.splitVertically()
	g.active = g.active.top
	g.Resize()
}

func (g *editor) killActiveView() {
	p := g.active.parent
	if p == nil {
		return
	}

	pp := p.parent
	sib := g.active.sibling()
	g.active.leaf.deactivate()
	g.active.leaf.detach()

	*p = *sib
	p.parent = pp
	p.reparent()

	g.active = p.firstLeafNode()
	g.active.leaf.activate()
	g.Resize()
}

func (g *editor) killAllViewsButActive() {
	g.views.traverse(func(v *viewTree) {
		if v == g.active {
			return
		}
		if v.leaf != nil {
			v.leaf.detach()
		}
	})
	g.views = g.active
	g.views.parent = nil
	g.Resize()
}

// Call it manually only when views layout has changed.
func (g *editor) Resize() {
	g.uiBuf = tulib.TermboxBuffer()
	views_area := g.uiBuf.Rect
	views_area.Height -= 1 // reserve space for command line
	g.views.resize(views_area)
}

func (g *editor) Draw() {
	var needsCursor bool
	if g.overlay != nil {
		needsCursor = g.overlay.needsCursor()
	}

	// draw everything
	g.views.draw()
	g.compositeRecursively(g.views)
	g.fixEdges(g.views)
	g.drawStatus(g.statusBuf.Bytes())

	// draw overlay if any
	if g.overlay != nil {
		g.overlay.draw()
	}

	// update cursor position
	var cx, cy int
	if needsCursor {
		// this can be true, only when g.Overlay != nil, see above
		cx, cy = g.overlay.cursorPosition()
	} else {
		cx, cy = g.CursorPosition()
	}
	termbox.SetCursor(cx, cy)
}

func (g *editor) drawStatus(text []byte) {
	lp := tulib.DefaultLabelParams
	r := g.uiBuf.Rect
	r.Y = r.Height - 1
	r.Height = 1
	g.uiBuf.Fill(r, termbox.Cell{Fg: lp.Fg, Bg: lp.Bg, Ch: ' '})
	g.uiBuf.DrawLabel(r, &lp, text)
}

func (g *editor) compositeRecursively(v *viewTree) {
	if v.leaf != nil {
		g.uiBuf.Blit(v.Rect, 0, 0, &v.leaf.uiBuf)
		return
	}

	if v.left != nil {
		g.compositeRecursively(v.left)
		g.compositeRecursively(v.right)
		splitter := v.right.Rect
		splitter.X -= 1
		splitter.Width = 1
		g.uiBuf.Fill(splitter, termbox.Cell{
			Fg: termbox.AttrReverse,
			Bg: termbox.AttrReverse,
			Ch: '│',
		})
		g.uiBuf.Set(splitter.X, splitter.Y+splitter.Height-1,
			termbox.Cell{
				Fg: termbox.AttrReverse,
				Bg: termbox.AttrReverse,
				Ch: '┴',
			})
	} else {
		g.compositeRecursively(v.top)
		g.compositeRecursively(v.bottom)
	}
}

func (g *editor) fixEdges(v *viewTree) {
	var x, y int
	var cell *termbox.Cell
	if v.leaf != nil {
		y = v.Y + v.Height - 1
		x = v.X - 1
		cell = g.uiBuf.Get(x, y)
		if cell != nil {
			switch cell.Ch {
			case '│':
				cell.Ch = '├'
			case '┤':
				cell.Ch = '┼'
			}
		}
		x = v.X + v.Width
		cell = g.uiBuf.Get(x, y)
		if cell != nil {
			switch cell.Ch {
			case '│':
				cell.Ch = '┤'
			case '├':
				cell.Ch = '┼'
			}
		}
		return
	}

	if v.left != nil {
		x = v.right.X - 1
		y = v.right.Y - 1
		cell = g.uiBuf.Get(x, y)
		if cell != nil {
			switch cell.Ch {
			case '─':
				cell.Ch = '┬'
			case '┴':
				cell.Ch = '┼'
			}
		}
		g.fixEdges(v.left)
		g.fixEdges(v.right)
	} else {
		g.fixEdges(v.top)
		g.fixEdges(v.bottom)
	}
}

// cursorPosition returns the absolute screen coordinates of the cursor
func (g *editor) CursorPosition() (int, int) {
	x, y := g.active.leaf.cursorPosition()
	return g.active.X + x, g.active.Y + y
}

func (g *editor) onSysKey(ev *termbox.Event) {
	switch ev.Key {
	case termbox.KeyCtrlQ:
		g.quit()
	case termbox.KeyCtrlZ:
		suspend(g)
	}
}

// Loop starts the editor main loop which consumes events from g.Events
func (e *editor) Loop() error {
	for ev := range e.Events {

		// The consume loop handles the event and any other events that
		// until there are no more in the queue.
	consume:
		for {
			if err := e.handleEvent(&ev); err != nil {
				return err
			}
			select {
			case nextEv := <-e.Events:
				ev = nextEv
			default:
				break consume
			}
		}

		e.Draw()
		termbox.Flush()
	}
	return nil
}

func (g *editor) handleEvent(ev *termbox.Event) error {
	switch ev.Type {
	case termbox.EventKey:
		g.setStatus("") // reset status on every key event
		g.onSysKey(ev)
		g.mode.onKey(ev)

		if g.quitFlag {
			return ErrQuit
		}
	case termbox.EventResize:
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		g.Resize()
		if g.overlay != nil {
			g.overlay.onResize(ev)
		}
	case termbox.EventError:
		return ev.Err
	}

	return nil
}

func (g *editor) setMode(m editorMode) {
	if g.mode != nil {
		g.mode.exit()
	}
	g.mode = m
	g.overlay = nil
	// Some modes can be overlays.
	if o, ok := m.(Overlay); ok {
		g.overlay = o
	}
}

func (g *editor) viewContext() viewContext {
	return viewContext{
		setStatus: func(f string, args ...interface{}) {
			g.setStatus(f, args...)
		},
		killBuffer: &g.killBuffer_,
		buffers:    &g.buffers,
	}
}

func (g *editor) hasUnsavedBuffers() bool {
	for _, buf := range g.buffers {
		if !buf.syncedWithDisk() {
			return true
		}
	}
	return false
}
