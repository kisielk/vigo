package editor

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kisielk/vigo/buffer"
	"github.com/kisielk/vigo/view"
	"github.com/nsf/termbox-go"
	"github.com/nsf/tulib"
)

type Mode interface {
	Enter(e *Editor)
	OnKey(ev *termbox.Event)
	Exit()
	Reset()
}

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
	uiBuf       tulib.Buffer
	active      *view.Tree // this one is always a leaf node
	views       *view.Tree // a root node
	buffers     []*buffer.Buffer
	statusBuf   bytes.Buffer
	quitFlag    bool
	killBuffer_ []byte

	LastSearchTerm string

	// Event channels
	UIEvents chan termbox.Event
	Commands chan Command
	redraw   chan struct{}

	cutBuffers *cutBuffers

	mode    Mode
	overlay Overlay
}

func (e *Editor) ActiveView() *view.View {
	return e.active.Leaf()
}

func (e *Editor) ActiveViewNode() *view.Tree {
	return e.active
}

func (e *Editor) Quit() {
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
		e.NewBufferFromFile(filename)
	}
	if len(e.buffers) == 0 {
		buf := buffer.NewEmptyBuffer()
		buf.Name = e.bufferName("unnamed")
		e.buffers = append(e.buffers, buf)
	}
	e.redraw = make(chan struct{})
	e.views = view.NewTree(view.NewView(e.viewContext(), e.buffers[0], e.redraw))
	e.active = e.views
	e.UIEvents = make(chan termbox.Event, 20)
	e.Commands = make(chan Command, 20)
	return e
}

func (e *Editor) findBufferByFullPath(path string) *buffer.Buffer {
	for _, buf := range e.buffers {
		if buf.Path == path {
			return buf
		}
	}
	return nil
}

// GetBuffer returns a buffer by name, or nil if there is no such buffer
func (e *Editor) getBuffer(name string) *buffer.Buffer {
	for _, buf := range e.buffers {
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

func (e *Editor) NewBufferFromFile(filename string) (*buffer.Buffer, error) {
	fullpath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("couldn't determine absolute path: %s", err)
	}
	buf := e.findBufferByFullPath(fullpath)
	if buf != nil {
		return buf, nil
	}

	f, err := os.Open(fullpath)
	if err == os.ErrNotExist {
		// Assume a new file
		e.SetStatus("(New file)")
		buf = buffer.NewEmptyBuffer()
	} else if err != nil {
		e.SetStatus(err.Error())
		return nil, err
	}
	defer f.Close()
	buf, err = buffer.NewBuffer(f)
	if err != nil {
		e.SetStatus(err.Error())
		return nil, err
	}
	buf.Path = fullpath

	buf.Name = e.bufferName(filename)
	e.buffers = append(e.buffers, buf)
	return buf, nil
}

func (e *Editor) SetStatus(format string, args ...interface{}) {
	e.statusBuf.Reset()
	fmt.Fprintf(&e.statusBuf, format, args...)
}

func (e *Editor) SetActiveViewNode(node *view.Tree) {
	e.active = node
}

func (e *Editor) SplitVertically() {
	if e.active.Width == 0 {
		return
	}
	e.active.SplitVertically()
	e.active = e.active.Left()
	e.Resize()
}

func (e *Editor) SplitHorizontally() {
	if e.active.Height == 0 {
		return
	}
	e.active.SplitHorizontally()
	e.active = e.active.Top()
	e.Resize()
}

func (e *Editor) killActiveView() {
	p := e.active.Parent()
	if p == nil {
		return
	}

	pp := p.Parent()
	sib := e.active.Sibling()
	e.active.Leaf().Detach()

	*p = *sib
	p.Reparent(pp)

	e.active = p.FirstLeafNode()
	e.Resize()
}

func (e *Editor) killAllViewsButActive() {
	e.views.Walk(func(v *view.Tree) {
		if v == e.active {
			return
		}
		if leaf := v.Leaf(); leaf != nil {
			leaf.Detach()
		}
	})
	e.views = e.active
	e.views.SetParent(nil)
	e.Resize()
}

// Call it manually only when views layout has changed.
func (e *Editor) Resize() {
	e.uiBuf = tulib.TermboxBuffer()
	viewsArea := e.uiBuf.Rect
	viewsArea.Height -= 1 // reserve space for command line
	e.views.Resize(viewsArea)
}

func (e *Editor) Draw() {
	var needsCursor bool
	if e.overlay != nil {
		needsCursor = e.overlay.NeedsCursor()
	}

	// draw everything
	e.views.Draw()
	e.compositeRecursively(e.views)
	e.fixEdges(e.views)
	e.DrawStatus(e.statusBuf.Bytes())

	// draw overlay if any
	if e.overlay != nil {
		e.overlay.Draw()
	}

	// update cursor position
	var cx, cy int
	if needsCursor {
		// this can be true, only when g.Overlay != nil, see above
		cx, cy = e.overlay.CursorPosition()
	} else {
		cx, cy = e.CursorPosition()
	}
	termbox.SetCursor(cx, cy)
}

func (e *Editor) DrawStatus(text []byte) {
	lp := tulib.DefaultLabelParams
	r := e.uiBuf.Rect
	r.Y = r.Height - 1
	r.Height = 1
	e.uiBuf.Fill(r, termbox.Cell{Fg: lp.Fg, Bg: lp.Bg, Ch: ' '})
	e.uiBuf.DrawLabel(r, &lp, text)
}

func (e *Editor) compositeRecursively(v *view.Tree) {
	if leaf := v.Leaf(); leaf != nil {
		buf := v.Leaf().UIBuf()
		e.uiBuf.Blit(v.Rect, 0, 0, &buf)
		return
	}

	if left := v.Left(); left != nil {
		e.compositeRecursively(left)
		right := v.Right()
		e.compositeRecursively(right)
		splitter := right.Rect
		splitter.X -= 1
		splitter.Width = 1
		uiBuf := e.uiBuf
		uiBuf.Fill(splitter, termbox.Cell{
			Fg: termbox.AttrReverse,
			Bg: termbox.AttrReverse,
			Ch: '│',
		})
		uiBuf.Set(splitter.X, splitter.Y+splitter.Height-1,
			termbox.Cell{
				Fg: termbox.AttrReverse,
				Bg: termbox.AttrReverse,
				Ch: '┴',
			})
	} else {
		e.compositeRecursively(v.Top())
		e.compositeRecursively(v.Bottom())
	}
}

func (e *Editor) fixEdges(v *view.Tree) {
	var x, y int
	var cell *termbox.Cell
	if leaf := v.Leaf(); leaf != nil {
		y = v.Y + v.Height - 1
		x = v.X - 1
		cell = e.uiBuf.Get(x, y)
		if cell != nil {
			switch cell.Ch {
			case '│':
				cell.Ch = '├'
			case '┤':
				cell.Ch = '┼'
			}
		}
		x = v.X + v.Width
		cell = e.uiBuf.Get(x, y)
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

	if left := v.Left(); left != nil {
		right := v.Right()
		x = right.X - 1
		y = right.Y - 1
		cell = e.uiBuf.Get(x, y)
		if cell != nil {
			switch cell.Ch {
			case '─':
				cell.Ch = '┬'
			case '┴':
				cell.Ch = '┼'
			}
		}
		e.fixEdges(left)
		e.fixEdges(right)
	} else {
		e.fixEdges(v.Top())
		e.fixEdges(v.Bottom())
	}
}

func (e *Editor) Height() int {
	return e.uiBuf.Height
}

// cursorPosition returns the absolute screen coordinates of the cursor
func (e *Editor) CursorPosition() (int, int) {
	x, y := e.active.Leaf().CursorPosition()
	return e.active.X + x, e.active.Y + y
}

func (e *Editor) onSysKey(ev *termbox.Event) {
	switch ev.Key {
	case termbox.KeyCtrlQ:
		e.Quit()
	case termbox.KeyCtrlZ:
		suspend(e)
	}
}

// Loops starts the editor main loop
func (e *Editor) Loop() error {
	for {
		select {
		case ev := <-e.UIEvents:
			// The consume loop handles the event and any other events that
			// until there are no more in the queue.
		consume:
			for {
				if err := e.handleUIEvent(&ev); err != nil {
					return err
				}
				select {
				case nextEv := <-e.UIEvents:
					ev = nextEv
				default:
					break consume
				}
			}
		case command := <-e.Commands:
			command.Apply(e)
		case <-e.redraw:
		}
		e.Draw()
		termbox.Flush()
	}
}

func (e *Editor) handleUIEvent(ev *termbox.Event) error {
	switch ev.Type {
	case termbox.EventKey:
		e.SetStatus("") // reset status on every key event
		e.onSysKey(ev)
		e.mode.OnKey(ev)

		if e.quitFlag {
			return ErrQuit
		}
	case termbox.EventResize:
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		e.Resize()
		if e.overlay != nil {
			e.overlay.OnResize(ev)
		}
	case termbox.EventError:
		return ev.Err
	}

	return nil
}

// SetMode sets active editor mode.
// The specified mode instance will react to keys and other user input until
// another mode is set.
func (e *Editor) SetMode(m *Mode) {
	if e.mode != nil {
		e.mode.Exit()
	}
	e.mode = *m
	e.overlay = nil
	// Some modes can be overlays.
	if o, ok := (*m).(Overlay); ok {
		e.overlay = o
	}
	(*m).Enter(e)
}

func (e *Editor) viewContext() view.Context {
	return view.NewContext(e.SetStatus, &e.killBuffer_, &e.buffers)
}

func (e *Editor) hasUnsavedBuffers() bool {
	for _, buf := range e.buffers {
		if !buf.SyncedWithDisk() {
			return true
		}
	}
	return false
}
