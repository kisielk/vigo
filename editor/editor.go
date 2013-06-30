package editor

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/kisielk/vigo/buffer"
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

type Command interface {
	Apply(*Editor)
}

type Editor struct {
	uiBuf        tulib.Buffer
	active       *viewTree // this one is always a leaf node
	views        *viewTree // a root node
	buffers      []*buffer.Buffer
	lastCmdClass vCommandClass
	statusBuf    bytes.Buffer
	quitFlag     bool
	killBuffer_  []byte

	// Event channels
	Events   chan termbox.Event
	Commands chan Command
	redraw   chan struct{}

	cutBuffers *cutBuffers

	mode    editorMode
	overlay Overlay
}

func (e *Editor) ActiveView() *view {
	return e.active.leaf
}

func (e *Editor) quit() {
	e.SetStatus("Quit")
	// Signals event loop to quit on next iteration.
	e.quitFlag = true
}

func NewEditor(filenames []string) *Editor {
	e := new(Editor)
	e.buffers = make([]*buffer.Buffer, 0, 20)
	e.cutBuffers = newCutBuffers()

	for _, filename := range filenames {
		//TODO: Check errors here
		e.newBufferFromFile(filename)
	}
	if len(e.buffers) == 0 {
		buf := buffer.NewEmptyBuffer()
		buf.Name = e.bufferName("unnamed")
		e.buffers = append(e.buffers, buf)
	}
	e.redraw = make(chan struct{})
	e.views = newViewTreeLeaf(nil, newView(e.viewContext(), e.buffers[0], e.redraw))
	e.active = e.views
	e.setMode(newNormalMode(e))
	e.Events = make(chan termbox.Event, 20)
	e.Commands = make(chan Command)
	return e
}

func (g *Editor) findBufferByFullPath(path string) *buffer.Buffer {
	for _, buf := range g.buffers {
		if buf.Path == path {
			return buf
		}
	}
	return nil
}

// GetBuffer returns a buffer by name, or nil if there is no such buffer
func (g *Editor) getBuffer(name string) *buffer.Buffer {
	for _, buf := range g.buffers {
		if buf.Name == name {
			return buf
		}
	}
	return nil
}

// BufferName generates a buffer name based on the one given.
func (e *Editor) bufferName(name string) string {
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

func (g *Editor) newBufferFromFile(filename string) (*buffer.Buffer, error) {
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
		g.SetStatus("(New file)")
		buf = buffer.NewEmptyBuffer()
	} else if err != nil {
		g.SetStatus(err.Error())
		return nil, err
	}
	defer f.Close()
	buf, err = buffer.NewBuffer(f)
	if err != nil {
		g.SetStatus(err.Error())
		return nil, err
	}
	buf.Path = fullpath

	buf.Name = g.bufferName(filename)
	g.buffers = append(g.buffers, buf)
	return buf, nil
}

func (g *Editor) SetStatus(format string, args ...interface{}) {
	g.statusBuf.Reset()
	fmt.Fprintf(&g.statusBuf, format, args...)
}

func (g *Editor) splitHorizontally() {
	if g.active.Width == 0 {
		return
	}
	g.active.splitHorizontally()
	g.active = g.active.left
	g.Resize()
}

func (g *Editor) splitVertically() {
	if g.active.Height == 0 {
		return
	}
	g.active.splitVertically()
	g.active = g.active.top
	g.Resize()
}

func (g *Editor) killActiveView() {
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

func (g *Editor) killAllViewsButActive() {
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
func (g *Editor) Resize() {
	g.uiBuf = tulib.TermboxBuffer()
	views_area := g.uiBuf.Rect
	views_area.Height -= 1 // reserve space for command line
	g.views.resize(views_area)
}

func (g *Editor) Draw() {
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

func (g *Editor) drawStatus(text []byte) {
	lp := tulib.DefaultLabelParams
	r := g.uiBuf.Rect
	r.Y = r.Height - 1
	r.Height = 1
	g.uiBuf.Fill(r, termbox.Cell{Fg: lp.Fg, Bg: lp.Bg, Ch: ' '})
	g.uiBuf.DrawLabel(r, &lp, text)
}

func (g *Editor) compositeRecursively(v *viewTree) {
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

func (g *Editor) fixEdges(v *viewTree) {
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
func (g *Editor) CursorPosition() (int, int) {
	x, y := g.active.leaf.cursorPosition()
	return g.active.X + x, g.active.Y + y
}

func (g *Editor) onSysKey(ev *termbox.Event) {
	switch ev.Key {
	case termbox.KeyCtrlQ:
		g.quit()
	case termbox.KeyCtrlZ:
		suspend(g)
	}
}

// Loop starts the editor main loop which consumes events from g.Events
func (e *Editor) Loop() error {
	for {
		select {
		case ev := <-e.Events:
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
		case command := <-e.Commands:
			// XXX: This causes a deadlock right now.
			command.Apply(e)
		case <-e.redraw:
		}
		e.Draw()
		termbox.Flush()
	}
	return nil
}

func (g *Editor) handleEvent(ev *termbox.Event) error {
	switch ev.Type {
	case termbox.EventKey:
		g.SetStatus("") // reset status on every key event
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

func (g *Editor) setMode(m editorMode) {
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

func (g *Editor) viewContext() viewContext {
	return viewContext{
		setStatus: func(f string, args ...interface{}) {
			g.SetStatus(f, args...)
		},
		killBuffer: &g.killBuffer_,
		buffers:    &g.buffers,
	}
}

func (g *Editor) hasUnsavedBuffers() bool {
	for _, buf := range g.buffers {
		if !buf.SyncedWithDisk() {
			return true
		}
	}
	return false
}
