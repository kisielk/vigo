package editor

import (
	"bytes"
	"fmt"
	"github.com/kisielk/vigo/buffer"
	"github.com/kisielk/vigo/utils"
	"github.com/nsf/termbox-go"
	"github.com/nsf/tulib"
	"os"
	"unicode/utf8"
)

//----------------------------------------------------------------------------
// dirty flag
//----------------------------------------------------------------------------

type dirtyFlag int

const (
	// dirtyContents indicates that the contents of the views buffer has been modified
	dirtyContents dirtyFlag = (1 << iota)

	// dirtyStatus indicates that the status line of the view needs to be updated
	dirtyStatus

	// diretyEverything indicates that everything needs to be updated
	dirtyEverything = dirtyContents | dirtyStatus
)

//----------------------------------------------------------------------------
// view location
//
// This structure represents a view location in the buffer. It needs to be
// separated from the view, because it's also being saved by the buffer (in case
// if at the moment buffer has no views attached to it).
//----------------------------------------------------------------------------

type viewLocation struct {
	cursor     buffer.Cursor
	topLine    *buffer.Line
	topLineNum int

	// Various cursor offsets from the beginning of the line:
	// 1. in characters
	// 2. in visual cells
	// An example would be the '\t' character, which gives 1 character
	// offset, but 'tabstop_length' visual cells offset.
	cursorCoffset int
	cursorVoffset int

	// This offset is different from these three above, because it's the
	// amount of visual cells you need to skip, before starting to show the
	// contents of the cursor line. The value stays as long as the cursor is
	// within the same line. When cursor jumps from one line to another, the
	// value is recalculated. The logic behind this variable is somewhat
	// close to the one behind the 'top_line' variable.
	lineVoffset int

	// this one is used for choosing the best location while traversing
	// vertically, every time 'cursor_voffset' changes due to horizontal
	// movement, this one must be changed as well
	lastCursorVoffset int
}

//----------------------------------------------------------------------------
// byte_range
//----------------------------------------------------------------------------

type byteRange struct {
	begin int
	end   int
}

func (r byteRange) includes(offset int) bool {
	return r.begin <= offset && r.end > offset
}

const hlFG = termbox.ColorCyan
const hlBG = termbox.ColorBlue

//----------------------------------------------------------------------------
// view tags
//----------------------------------------------------------------------------

type viewTag struct {
	begLine   int
	begOffset int
	endLine   int
	endOffset int
	fg        termbox.Attribute
	bg        termbox.Attribute
}

func (t *viewTag) includes(line, offset int) bool {
	if line < t.begLine || line > t.endLine {
		return false
	}
	if line == t.begLine && offset < t.begOffset {
		return false
	}
	if line == t.endLine && offset >= t.endOffset {
		return false
	}
	return true
}

var defaultViewTag = viewTag{
	fg: termbox.ColorDefault,
	bg: termbox.ColorDefault,
}

//----------------------------------------------------------------------------
// view context
//----------------------------------------------------------------------------

type viewContext struct {
	setStatus  func(format string, args ...interface{})
	killBuffer *[]byte
	buffers    *[]*buffer.Buffer
}

//----------------------------------------------------------------------------
// view
//
// Think of it as a window. It draws contents from a portion of a buffer into
// 'uibuf' and maintains things like cursor position.
//----------------------------------------------------------------------------

type view struct {
	viewLocation
	ctx             viewContext
	buf             *buffer.Buffer // currently displayed buffer
	uiBuf           tulib.Buffer
	dirty           dirtyFlag
	highlightBytes  []byte
	highlightRanges []byteRange
	tags            []viewTag
	redraw          chan struct{}

	// statusBuf is a buffer used for drawing the status line
	statusBuf bytes.Buffer

	lastCommand viewCommand

	bufferEvents chan buffer.BufferEvent
}

func newView(ctx viewContext, buf *buffer.Buffer, redraw chan struct{}) *view {
	v := new(view)
	v.ctx = ctx
	v.uiBuf = tulib.NewBuffer(1, 1)
	v.attach(buf)
	v.highlightRanges = make([]byteRange, 0, 10)
	v.tags = make([]viewTag, 0, 10)
	v.redraw = redraw
	return v
}

func (v *view) activate() {
	v.lastCommand = viewCommand{Cmd: vCommandNone}
}

func (v *view) deactivate() {
}

func (v *view) attach(b *buffer.Buffer) {
	if v.buf == b {
		return
	}

	if v.buf != nil {
		v.detach()
	}
	v.buf = b
	v.viewLocation = viewLocation{
		topLine:    b.FirstLine,
		topLineNum: 1,
		cursor: buffer.Cursor{
			Line:    b.FirstLine,
			LineNum: 1,
		},
	}
	// Add a small message buffer, otherwise buffer methods
	// sending several consequitive events will lock up.
	v.bufferEvents = make(chan buffer.BufferEvent, 10)
	go func() {
		for e := range v.bufferEvents {
			switch e.Type {
			case buffer.BufferEventInsert:
				v.onInsertAdjustTopLine(e.Action)
				c := v.cursor
				c.OnInsertAdjust(e.Action)
				v.moveCursorTo(c)
				v.dirty = dirtyEverything
				// FIXME for unfocused views, just call onInsert
				// v.onInsert(e.Action)
			case buffer.BufferEventDelete:
				v.onDeleteAdjustTopLine(e.Action)
				c := v.cursor
				c.OnDeleteAdjust(e.Action)
				v.moveCursorTo(c)
				v.dirty = dirtyEverything
				// FIXME for unfocused views, just call onDelete
				// v.onDelete(e.Action)
			case buffer.BufferEventBOF:
				v.ctx.setStatus("Beginning of buffer")
				v.dirty |= dirtyStatus
			case buffer.BufferEventEOF:
				v.ctx.setStatus("End of buffer")
				v.dirty |= dirtyStatus
			case buffer.BufferEventHistoryBack:
				v.ctx.setStatus("Undo!")
				v.dirty |= dirtyStatus
			case buffer.BufferEventHistoryForward:
				v.ctx.setStatus("Redo!")
				v.dirty |= dirtyStatus
			case buffer.BufferEventHistoryStart:
				v.ctx.setStatus("No further undo information")
				v.dirty |= dirtyStatus
			case buffer.BufferEventHistoryEnd:
				v.ctx.setStatus("No further redo information")
				v.dirty |= dirtyStatus
			case buffer.BufferEventSave:
				v.dirty |= dirtyStatus
			}
			v.redraw <- struct{}{}
		}
	}()
	v.buf.AddListener(v.bufferEvents)
	v.dirty = dirtyEverything
}

func (v *view) detach() {
	// Stop listening to current buffer and close event loop.
	v.buf.RemoveListener(v.bufferEvents)
	close(v.bufferEvents)
	v.bufferEvents = nil
	v.buf = nil
}

// Resize the 'v.uibuf', adjusting things accordingly.
func (v *view) resize(w, h int) {
	v.uiBuf.Resize(w, h)
	v.adjustLineVoffset()
	v.adjustTopLine()
	v.dirty = dirtyEverything
}

func (v *view) height() int {
	return v.uiBuf.Height
}

func (v *view) verticalThreshold() int {
	maxVthreshold := (v.height() - 1) / 2
	if viewVerticalThreshold > maxVthreshold {
		return maxVthreshold
	}
	return viewVerticalThreshold
}

func (v *view) horizontalThreshold() int {
	max_h_threshold := (v.width() - 1) / 2
	if viewHorizontalThreshold > max_h_threshold {
		return max_h_threshold
	}
	return viewHorizontalThreshold
}

func (v *view) width() int {
	// TODO: perhaps if I want to draw line numbers, I will hack it there
	return v.uiBuf.Width
}

func (v *view) drawLine(line *buffer.Line, lineNum, coff, lineVoffset int) {
	x := 0
	tabstop := 0
	bx := 0
	data := line.Data

	if len(v.highlightBytes) > 0 {
		v.findHighlightRangesForLine(data)
	}
	for {
		rx := x - lineVoffset
		if len(data) == 0 {
			break
		}

		if x == tabstop {
			tabstop += tabstopLength
		}

		if rx >= v.uiBuf.Width {
			last := coff + v.uiBuf.Width - 1
			v.uiBuf.Cells[last] = termbox.Cell{
				Ch: '→',
				Fg: termbox.ColorDefault,
				Bg: termbox.ColorDefault,
			}
			break
		}

		r, rlen := utf8.DecodeRune(data)
		switch {
		case r == '\t':
			// fill with spaces to the next tabstop
			for ; x < tabstop; x++ {
				rx := x - lineVoffset
				if rx >= v.uiBuf.Width {
					break
				}

				if rx >= 0 {
					v.uiBuf.Cells[coff+rx] = v.makeCell(
						lineNum, bx, ' ')
				}
			}
		case r < 32:
			// invisible chars like ^R or ^@
			if rx >= 0 {
				v.uiBuf.Cells[coff+rx] = termbox.Cell{
					Ch: '^',
					Fg: termbox.ColorRed,
					Bg: termbox.ColorDefault,
				}
			}
			x++
			rx = x - lineVoffset
			if rx >= v.uiBuf.Width {
				break
			}
			if rx >= 0 {
				v.uiBuf.Cells[coff+rx] = termbox.Cell{
					Ch: utils.InvisibleRuneTable[r],
					Fg: termbox.ColorRed,
					Bg: termbox.ColorDefault,
				}
			}
			x++
		default:
			if rx >= 0 {
				v.uiBuf.Cells[coff+rx] = v.makeCell(
					lineNum, bx, r)
			}
			x++
		}
		data = data[rlen:]
		bx += rlen
	}

	if lineVoffset != 0 {
		v.uiBuf.Cells[coff] = termbox.Cell{
			Ch: '←',
			Fg: termbox.ColorDefault,
			Bg: termbox.ColorDefault,
		}
	}
}

func (v *view) drawContents() {
	if len(v.highlightBytes) == 0 {
		v.highlightRanges = v.highlightRanges[:0]
	}

	// clear the buffer
	v.uiBuf.Fill(v.uiBuf.Rect, termbox.Cell{
		Ch: ' ',
		Fg: termbox.ColorDefault,
		Bg: termbox.ColorDefault,
	})

	if v.uiBuf.Width == 0 || v.uiBuf.Height == 0 {
		return
	}

	// draw lines
	line := v.topLine
	coff := 0
	for y, h := 0, v.height(); y < h; y++ {
		if line == nil {
			break
		}

		if line == v.cursor.Line {
			// special case, cursor line
			v.drawLine(line, v.topLineNum+y, coff, v.lineVoffset)
		} else {
			v.drawLine(line, v.topLineNum+y, coff, 0)
		}

		coff += v.uiBuf.Width
		line = line.Next
	}
}

func (v *view) drawStatus() {
	// fill background with '─'
	lp := tulib.DefaultLabelParams
	lp.Bg = termbox.AttrReverse
	lp.Fg = termbox.AttrReverse | termbox.AttrBold
	v.uiBuf.Fill(
		tulib.Rect{X: 0, Y: v.height(), Width: v.uiBuf.Width, Height: 1},
		termbox.Cell{Fg: termbox.AttrReverse, Bg: termbox.AttrReverse, Ch: '─'},
	)

	// on disk sync status
	if !v.buf.SyncedWithDisk() {
		cell := termbox.Cell{
			Fg: termbox.AttrReverse,
			Bg: termbox.AttrReverse,
			Ch: '*',
		}
		v.uiBuf.Set(1, v.height(), cell)
		v.uiBuf.Set(2, v.height(), cell)
	}

	// filename
	fmt.Fprintf(&v.statusBuf, "  %s  ", v.buf.Name)
	v.uiBuf.DrawLabel(tulib.Rect{X: 5, Y: v.height(), Width: v.uiBuf.Width, Height: 1}, &lp, v.statusBuf.Bytes())
	namel := v.statusBuf.Len()
	lp.Fg = termbox.AttrReverse
	v.statusBuf.Reset()
	fmt.Fprintf(&v.statusBuf, "(%d, %d)  ", v.cursor.LineNum, v.cursorVoffset)
	v.uiBuf.DrawLabel(tulib.Rect{X: 5 + namel, Y: v.height(), Width: v.uiBuf.Width, Height: 1}, &lp, v.statusBuf.Bytes())
	v.statusBuf.Reset()
}

// Draw the current view to the 'v.uibuf'.
func (v *view) draw() {
	if v.dirty&dirtyContents != 0 {
		v.dirty &^= dirtyContents
		v.drawContents()
	}

	if v.dirty&dirtyStatus != 0 {
		v.dirty &^= dirtyStatus
		v.drawStatus()
	}
}

// Center view on the cursor.
func (v *view) centerViewOnCursor() {
	v.topLine = v.cursor.Line
	v.topLineNum = v.cursor.LineNum
	v.moveTopLineNtimes(-v.height() / 2)
	v.dirty = dirtyEverything
}

func (v *view) moveCursorToLine(n int) {
	v.moveCursorBeginningOfFile()
	v.moveCursorLineNtimes(n - 1)
	v.centerViewOnCursor()
}

// Move top line 'n' times forward or backward.
func (v *view) moveTopLineNtimes(n int) {
	if n == 0 {
		return
	}

	top := v.topLine
	for top.Prev != nil && n < 0 {
		top = top.Prev
		v.topLineNum--
		n++
	}
	for top.Next != nil && n > 0 {
		top = top.Next
		v.topLineNum++
		n--
	}
	v.topLine = top
}

// Move cursor line 'n' times forward or backward.
func (v *view) moveCursorLineNtimes(n int) {
	if n == 0 {
		return
	}

	cursor := v.cursor.Line
	for cursor.Prev != nil && n < 0 {
		cursor = cursor.Prev
		v.cursor.LineNum--
		n++
	}
	for cursor.Next != nil && n > 0 {
		cursor = cursor.Next
		v.cursor.LineNum++
		n--
	}
	v.cursor.Line = cursor
}

// When 'top_line' was changed, call this function to possibly adjust the
// 'cursor_line'.
func (v *view) adjustCursorLine() {
	vt := v.verticalThreshold()
	cursor := v.cursor.Line
	co := v.cursor.LineNum - v.topLineNum
	h := v.height()

	if cursor.Next != nil && co < vt {
		v.moveCursorLineNtimes(vt - co)
	}

	if cursor.Prev != nil && co >= h-vt {
		v.moveCursorLineNtimes((h - vt) - co - 1)
	}

	if cursor != v.cursor.Line {
		cursor = v.cursor.Line
		bo, co, vo := cursor.FindClosestOffsets(v.lastCursorVoffset)
		v.cursor.Boffset = bo
		v.cursorCoffset = co
		v.cursorVoffset = vo
		v.lineVoffset = 0
		v.adjustLineVoffset()
		v.dirty = dirtyEverything
	}
}

// When 'cursor_line' was changed, call this function to possibly adjust the
// 'top_line'.
func (v *view) adjustTopLine() {
	vt := v.verticalThreshold()
	top := v.topLine
	co := v.cursor.LineNum - v.topLineNum
	h := v.height()

	if top.Next != nil && co >= h-vt {
		v.moveTopLineNtimes(co - (h - vt) + 1)
		v.dirty = dirtyEverything
	}

	if top.Prev != nil && co < vt {
		v.moveTopLineNtimes(co - vt)
		v.dirty = dirtyEverything
	}
}

// When 'cursor_voffset' was changed usually > 0, then call this function to
// possibly adjust 'line_voffset'.
func (v *view) adjustLineVoffset() {
	ht := v.horizontalThreshold()
	w := v.uiBuf.Width
	vo := v.lineVoffset
	cvo := v.cursorVoffset
	threshold := w - 1
	if vo != 0 {
		threshold = w - ht
	}

	if cvo-vo >= threshold {
		vo = cvo + (ht - w + 1)
	}

	if vo != 0 && cvo-vo < ht {
		vo = cvo - ht
		if vo < 0 {
			vo = 0
		}
	}

	if v.lineVoffset != vo {
		v.lineVoffset = vo
		v.dirty = dirtyEverything
	}
}

func (v *view) cursorPosition() (int, int) {
	y := v.cursor.LineNum - v.topLineNum
	x := v.cursorVoffset - v.lineVoffset
	return x, y
}

func (v *view) cursorPositionFor(cursor buffer.Cursor) (int, int) {
	y := cursor.LineNum - v.topLineNum
	vo, _ := cursor.VoffsetCoffset()
	x := vo - v.lineVoffset
	return x, y
}

// Move cursor to the 'boffset' position in the 'line'. Obviously 'line' must be
// from the attached buffer. If 'boffset' < 0, use 'last_cursor_voffset'. Keep
// in mind that there is no need to maintain connections between lines (e.g. for
// moving from a deleted line to another line).
func (v *view) moveCursorTo(c buffer.Cursor) {
	v.dirty |= dirtyStatus
	if c.Boffset < 0 {
		bo, co, vo := c.Line.FindClosestOffsets(v.lastCursorVoffset)
		v.cursor.Boffset = bo
		v.cursorCoffset = co
		v.cursorVoffset = vo
	} else {
		vo, co := c.VoffsetCoffset()
		v.cursor.Boffset = c.Boffset
		v.cursorCoffset = co
		v.cursorVoffset = vo
	}

	if c.Boffset >= 0 {
		v.lastCursorVoffset = v.cursorVoffset
	}

	if c.Line != v.cursor.Line {
		if v.lineVoffset != 0 {
			v.dirty = dirtyEverything
		}
		v.lineVoffset = 0
	}
	v.cursor.Line = c.Line
	v.cursor.LineNum = c.LineNum
	v.adjustLineVoffset()
	v.adjustTopLine()
}

// Move cursor one character forward.
func (v *view) moveCursorForward() {
	c := v.cursor
	if c.LastLine() && c.EOL() {
		v.ctx.setStatus("End of buffer")
		return
	}

	c.NextRune(configWrapRight)
	v.moveCursorTo(c)
}

// Move cursor one character backward.
func (v *view) moveCursorBackward() {
	c := v.cursor
	if c.FirstLine() && c.BOL() {
		v.ctx.setStatus("Beginning of buffer")
		return
	}

	c.PrevRune(configWrapLeft)
	v.moveCursorTo(c)
}

// Move cursor to the next line.
func (v *view) moveCursorNextLine() {
	c := v.cursor
	if !c.LastLine() {
		c = buffer.Cursor{
			Line:    c.Line.Next,
			LineNum: c.LineNum + 1,
			Boffset: -1,
		}
		v.moveCursorTo(c)
	} else {
		v.ctx.setStatus("End of buffer")
	}
}

// Move cursor to the previous line.
func (v *view) moveCursorPrevLine() {
	c := v.cursor
	if !c.FirstLine() {
		c = buffer.Cursor{
			Line:    c.Line.Prev,
			LineNum: c.LineNum - 1,
			Boffset: -1,
		}
		v.moveCursorTo(c)
	} else {
		v.ctx.setStatus("Beginning of buffer")
	}
}

// Move cursor to the beginning of the line.
func (v *view) moveCursorBeginningOfLine() {
	c := v.cursor
	c.MoveBOL()
	v.moveCursorTo(c)
}

// Move cursor to the end of the line.
func (v *view) moveCursorEndOfLine() {
	c := v.cursor
	c.MoveEOL()
	v.moveCursorTo(c)
}

// Move cursor to the beginning of the file (buffer).
func (v *view) moveCursorBeginningOfFile() {
	c := buffer.Cursor{
		Line:    v.buf.FirstLine,
		LineNum: 1,
		Boffset: 0,
	}
	v.moveCursorTo(c)
}

// Move cursor to the end of the file (buffer).
func (v *view) moveCursorEndOfFile() {
	c := buffer.Cursor{
		Line:    v.buf.LastLine,
		LineNum: v.buf.NumLines,
		Boffset: v.buf.LastLine.Len(),
	}
	v.moveCursorTo(c)
}

// Move cursor to the end of the next (or current) word.
func (v *view) moveCursorWordFoward() {
	c := v.cursor
	ok := c.NextWord()
	v.moveCursorTo(c)
	if !ok {
		v.ctx.setStatus("End of buffer")
	}
}

func (v *view) moveCursorWordBackward() {
	c := v.cursor
	ok := c.PrevWord()
	v.moveCursorTo(c)
	if !ok {
		v.ctx.setStatus("Beginning of buffer")
	}
}

// Move view 'n' lines forward or backward.
func (v *view) moveViewNlines(n int) {
	prevtop := v.topLineNum
	v.moveTopLineNtimes(n)
	if prevtop != v.topLineNum {
		v.adjustCursorLine()
		v.dirty = dirtyEverything
	}
}

// Check if it's possible to move view 'n' lines forward or backward.
func (v *view) canMoveTopLineNtimes(n int) bool {
	if n == 0 {
		return true
	}

	top := v.topLine
	for top.Prev != nil && n < 0 {
		top = top.Prev
		n++
	}
	for top.Next != nil && n > 0 {
		top = top.Next
		n--
	}

	if n != 0 {
		return false
	}
	return true
}

// Move view 'n' lines forward or backward only if it's possible.
func (v *view) maybeMoveViewNlines(n int) {
	if v.canMoveTopLineNtimes(n) {
		v.moveViewNlines(n)
	}
}

// If not at the EOL, remove contents of the current line from the cursor to the
// end. Otherwise behave like 'delete'.
func (v *view) killLine() {
	c := v.cursor
	if !c.EOL() {
		// kill data from the cursor to the EOL
		len := c.Line.Len() - c.Boffset
		v.appendToKillBuffer(c, len)
		v.buf.Delete(c, len)
		return
	}
	v.appendToKillBuffer(c, 1)
	v.buf.DeleteRune(c)
}

func (v *view) killWord() {
	c1 := v.cursor
	c2 := c1
	c2.NextWord()
	d := c1.Distance(c2)
	if d > 0 {
		v.appendToKillBuffer(c1, d)
		v.buf.Delete(c1, d)
	}
}

func (v *view) killWordBackward() {
	c2 := v.cursor
	c1 := c2
	c1.PrevWord()
	d := c1.Distance(c2)
	if d > 0 {
		v.prependToKillBuffer(c1, d)
		v.buf.Delete(c1, d)
		v.moveCursorTo(c1)
	}
}

func (v *view) onInsertAdjustTopLine(a *buffer.Action) {
	if a.Cursor.LineNum < v.topLineNum && len(a.Lines) > 0 {
		// inserted one or more lines above the view
		v.topLineNum += len(a.Lines)
		v.dirty |= dirtyStatus
	}
}

func (v *view) onDeleteAdjustTopLine(a *buffer.Action) {
	if a.Cursor.LineNum < v.topLineNum {
		// deletion above the top line
		if len(a.Lines) == 0 {
			return
		}

		topnum := v.topLineNum
		first, last := a.DeletedLines()
		if first <= topnum && topnum <= last {
			// deleted the top line, adjust the pointers
			if a.Cursor.Line.Next != nil {
				v.topLine = a.Cursor.Line.Next
				v.topLineNum = a.Cursor.LineNum + 1
			} else {
				v.topLine = a.Cursor.Line
				v.topLineNum = a.Cursor.LineNum
			}
			v.dirty = dirtyEverything
		} else {
			// no need to worry
			v.topLineNum -= len(a.Lines)
			v.dirty |= dirtyStatus
		}
	}
}

func (v *view) onInsert(a *buffer.Action) {
	v.onInsertAdjustTopLine(a)
	if v.topLineNum+v.height() <= a.Cursor.LineNum {
		// inserted something below the view, don't care
		return
	}
	if a.Cursor.LineNum < v.topLineNum {
		// inserted something above the top line
		if len(a.Lines) > 0 {
			// inserted one or more lines, adjust line numbers
			v.cursor.LineNum += len(a.Lines)
			v.dirty |= dirtyStatus
		}
		return
	}
	c := v.cursor
	c.OnInsertAdjust(a)
	v.moveCursorTo(c)
	v.lastCursorVoffset = v.cursorVoffset
	v.dirty = dirtyEverything
}

func (v *view) onDelete(a *buffer.Action) {
	v.onDeleteAdjustTopLine(a)
	if v.topLineNum+v.height() <= a.Cursor.LineNum {
		// deleted something below the view, don't care
		return
	}
	if a.Cursor.LineNum < v.topLineNum {
		// deletion above the top line
		if len(a.Lines) == 0 {
			return
		}

		_, last := a.DeletedLines()
		if last < v.topLineNum {
			// no need to worry
			v.cursor.LineNum -= len(a.Lines)
			v.dirty |= dirtyStatus
			return
		}
	}
	c := v.cursor
	c.OnDeleteAdjust(a)
	v.moveCursorTo(c)
	v.lastCursorVoffset = v.cursorVoffset
	v.dirty = dirtyEverything
}

func (v *view) onVcommand(c viewCommand) {
	lastClass := v.lastCommand.Cmd.class()
	if c.Cmd.class() != lastClass || lastClass == vCommandClassMisc {
		v.buf.FinalizeActionGroup()
	}

	reps := c.Reps
	if reps == 0 {
		reps = 1
	}

	for i := 0; i < reps; i++ {
		switch c.Cmd {
		case vCommandMoveCursorForward:
			v.moveCursorForward()

		case vCommandMoveCursorBackward:
			v.moveCursorBackward()

		case vCommandMoveCursorWordForward:
			v.moveCursorWordFoward()

		case vCommandMoveCursorWordBackward:
			v.moveCursorWordBackward()

		case vCommandMoveCursorNextLine:
			v.moveCursorNextLine()

		case vCommandMoveCursorPrevLine:
			v.moveCursorPrevLine()

		case vCommandMoveCursorBeginningOfLine:
			v.moveCursorBeginningOfLine()

		case vCommandMoveCursorEndOfLine:
			v.moveCursorEndOfLine()

		case vCommandMoveCursorBeginningOfFile:
			v.moveCursorBeginningOfFile()

		case vCommandMoveCursorEndOfFile:
			v.moveCursorEndOfFile()
			/*
				case vCommandMoveCursorToLine:
					v.moveCursorToLine(int(arg))
			*/

		case vCommandMoveViewHalfForward:
			v.maybeMoveViewNlines(v.height() / 2)
		case vCommandMoveViewHalfBackward:
			v.moveViewNlines(-v.height() / 2)
		case vCommandInsertRune:
			v.buf.InsertRune(v.cursor, c.Rune)
		case vCommandYank:
			v.yank()
		case vCommandDeleteRuneBackward:
			v.buf.DeleteRuneBackward(v.cursor)
		case vCommandDeleteRune:
			v.buf.DeleteRune(v.cursor)
		case vCommandKillLine:
			v.killLine()
		case vCommandKillWord:
			v.killWord()
		case vCommandKillWordBackward:
			v.killWordBackward()
		case vCommandUndo:
			v.buf.Undo()
		case vCommandRedo:
			v.buf.Redo()
		case vCommandWordToUpper:
			v.wordTo(bytes.ToUpper)
		case vCommandWordToTitle:
			v.wordTo(func(s []byte) []byte {
				return bytes.Title(bytes.ToLower(s))
			})
		case vCommandWordToLower:
			v.wordTo(bytes.ToLower)
		}
	}

	v.lastCommand = c
}

func (v *view) dumpInfo() {
	p := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}

	p("Top line num: %d\n", v.topLineNum)
}

func (v *view) findHighlightRangesForLine(data []byte) {
	v.highlightRanges = v.highlightRanges[:0]
	offset := 0
	for {
		i := bytes.Index(data, v.highlightBytes)
		if i == -1 {
			return
		}

		v.highlightRanges = append(v.highlightRanges, byteRange{
			begin: offset + i,
			end:   offset + i + len(v.highlightBytes),
		})
		data = data[i+len(v.highlightBytes):]
		offset += i + len(v.highlightBytes)
	}
}

func (v *view) inOneOfHighlightRanges(offset int) bool {
	for _, r := range v.highlightRanges {
		if r.includes(offset) {
			return true
		}
	}
	return false
}

func (v *view) tag(line, offset int) *viewTag {
	for i := range v.tags {
		t := &v.tags[i]
		if t.includes(line, offset) {
			return t
		}
	}
	return &defaultViewTag
}

func (v *view) makeCell(line, offset int, ch rune) termbox.Cell {
	tag := v.tag(line, offset)
	if tag != &defaultViewTag {
		return termbox.Cell{
			Ch: ch,
			Fg: tag.fg,
			Bg: tag.bg,
		}
	}

	cell := termbox.Cell{
		Ch: ch,
		Fg: tag.fg,
		Bg: tag.bg,
	}
	if v.inOneOfHighlightRanges(offset) {
		cell.Fg = hlFG
		cell.Bg = hlBG
	}
	return cell
}

func (v *view) appendToKillBuffer(cursor buffer.Cursor, nbytes int) {
	kb := *v.ctx.killBuffer

	switch v.lastCommand.Cmd {
	case vCommandKillWord, vCommandKillWordBackward, vCommandKillRegion, vCommandKillLine:
	default:
		kb = kb[:0]
	}

	kb = append(kb, cursor.ExtractBytes(nbytes)...)
	*v.ctx.killBuffer = kb
}

func (v *view) prependToKillBuffer(cursor buffer.Cursor, nbytes int) {
	kb := *v.ctx.killBuffer

	switch v.lastCommand.Cmd {
	case vCommandKillWord, vCommandKillWordBackward, vCommandKillRegion, vCommandKillLine:
	default:
		kb = kb[:0]
	}

	kb = append(cursor.ExtractBytes(nbytes), kb...)
	*v.ctx.killBuffer = kb
}

func (v *view) yank() {
	buf := *v.ctx.killBuffer
	cursor := v.cursor

	if len(buf) == 0 {
		return
	}
	cbuf := utils.CloneByteSlice(buf)
	v.buf.Insert(cursor, cbuf)
	for len(buf) > 0 {
		_, rlen := utf8.DecodeRune(buf)
		buf = buf[rlen:]
		cursor.NextRune(true)
	}
	v.moveCursorTo(cursor)
}

func (v *view) indentLine(line buffer.Cursor) {
	line.Boffset = 0
	v.buf.Insert(line, []byte{'\t'})
	if v.cursor.Line == line.Line {
		cursor := v.cursor
		cursor.Boffset += 1
		v.moveCursorTo(cursor)
	}
}

func (v *view) deindentLine(line buffer.Cursor) {
	line.Boffset = 0
	if r, _ := line.RuneUnder(); r == '\t' {
		v.buf.Delete(line, 1)
	}
	if v.cursor.Line == line.Line && v.cursor.Boffset > 0 {
		cursor := v.cursor
		cursor.Boffset -= 1
		v.moveCursorTo(cursor)
	}
}

func (v *view) wordTo(filter func([]byte) []byte) {
	c1, c2 := v.cursor, v.cursor
	c2.NextWord()
	v.filterText(c1, c2, filter)
	c1.NextWord()
	v.moveCursorTo(c1)
}

// Filter _must_ return a new slice and shouldn't touch contents of the
// argument, perfect filter examples are: bytes.Title, bytes.ToUpper,
// bytes.ToLower
func (v *view) filterText(from, to buffer.Cursor, filter func([]byte) []byte) {
	c1, c2 := buffer.SortCursors(from, to)
	d := c1.Distance(c2)
	v.buf.Delete(c1, d)
	data := filter(v.buf.History.LastAction().Data)
	v.buf.Insert(c1, data)
}

//----------------------------------------------------------------------------
// view commands
//----------------------------------------------------------------------------

type vCommandClass int

const (
	vCommandClassNone vCommandClass = iota
	vCommandClassMovement
	vCommandClassInsertion
	vCommandClassDeletion
	vCommandClassHistory
	vCommandClassMisc
)

type viewCommand struct {
	// The command to execute
	Cmd vCommand

	// Number of times to repeat the command
	Reps int

	// Rune to use in the command
	Rune rune
}

type vCommand int

const (
	vCommandNone vCommand = iota

	// movement commands (finalize undo action group)
	_vCommandMovementBeg
	vCommandMoveCursorForward
	vCommandMoveCursorBackward
	vCommandMoveCursorWordForward
	vCommandMoveCursorWordBackward
	vCommandMoveCursorNextLine
	vCommandMoveCursorPrevLine
	vCommandMoveCursorBeginningOfLine
	vCommandMoveCursorEndOfLine
	vCommandMoveCursorBeginningOfFile
	vCommandMoveCursorEndOfFile
	vCommandMoveCursorToLine
	vCommandMoveViewHalfForward
	vCommandMoveViewHalfBackward
	_vCommandMovementEnd

	// insertion commands
	_vCommandInsertionBeg
	vCommandInsertRune
	vCommandYank
	_vCommandInsertionEnd

	// deletion commands
	_vCommandDeletionBeg
	vCommandDeleteRuneBackward
	vCommandDeleteRune
	vCommandKillLine
	vCommandKillWord
	vCommandKillWordBackward
	vCommandKillRegion
	_vCommandDeletionEnd

	// history commands (undo/redo)
	_vCommandHistoryBeg
	vCommandUndo
	vCommandRedo
	_vCommandHistoryEnd

	// misc commands
	_vCommandMiscBeg
	vCommandIndentRegion
	vCommandDeindentRegion
	vCommandCopyRegion
	vCommandRegionToUpper
	vCommandRegionToLower
	vCommandWordToUpper
	vCommandWordToTitle
	vCommandWordToLower
	_vCommandMiscEnd
)

func (c vCommand) class() vCommandClass {
	switch {
	case c > _vCommandMovementBeg && c < _vCommandMovementEnd:
		return vCommandClassMovement
	case c > _vCommandInsertionBeg && c < _vCommandInsertionEnd:
		return vCommandClassInsertion
	case c > _vCommandDeletionBeg && c < _vCommandDeletionEnd:
		return vCommandClassDeletion
	case c > _vCommandHistoryBeg && c < _vCommandHistoryEnd:
		return vCommandClassHistory
	case c > _vCommandMiscBeg && c < _vCommandMiscEnd:
		return vCommandClassMisc
	}
	return vCommandClassNone
}
