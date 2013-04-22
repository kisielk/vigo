package main

import (
	"bytes"
	"fmt"
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
	DIRTY_CONTENTS dirtyFlag = (1 << iota)
	DIRTY_STATUS

	DIRTY_EVERYTHING = DIRTY_CONTENTS | DIRTY_STATUS
)

//----------------------------------------------------------------------------
// view location
//
// This structure represents a view location in the buffer. It needs to be
// separated from the view, because it's also being saved by the buffer (in case
// if at the moment buffer has no views attached to it).
//----------------------------------------------------------------------------

type viewLocation struct {
	cursor     cursor
	topLine    *line
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
	buffers    *[]*buffer
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
	tmpBuf          bytes.Buffer // temporary buffer for status bar text
	buf             *buffer      // currently displayed buffer
	uiBuf           tulib.Buffer
	dirty           dirtyFlag
	oneline         bool
	highlightBytes  []byte
	highlightRanges []byteRange
	tags            []viewTag

	lastCommand viewCommand
}

func newView(ctx viewContext, buf *buffer) *view {
	v := new(view)
	v.ctx = ctx
	v.uiBuf = tulib.NewBuffer(1, 1)
	v.attach(buf)
	v.highlightRanges = make([]byteRange, 0, 10)
	v.tags = make([]viewTag, 0, 10)
	return v
}

func (v *view) activate() {
	v.lastCommand = viewCommand{Cmd: vCommandNone}
}

func (v *view) deactivate() {
}

func (v *view) attach(b *buffer) {
	if v.buf == b {
		return
	}

	if v.buf != nil {
		v.detach()
	}
	v.buf = b
	v.viewLocation = b.loc
	b.addView(v)
	v.dirty = DIRTY_EVERYTHING
}

func (v *view) detach() {
	v.buf.deleteView(v)
	v.buf = nil
}

// Resize the 'v.uibuf', adjusting things accordingly.
func (v *view) resize(w, h int) {
	v.uiBuf.Resize(w, h)
	v.adjustLineVoffset()
	v.adjustTopLine()
	v.dirty = DIRTY_EVERYTHING
}

func (v *view) height() int {
	if !v.oneline {
		return v.uiBuf.Height - 1
	}
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

func (v *view) drawLine(line *line, lineNum, coff, lineVoffset int) {
	x := 0
	tabstop := 0
	bx := 0
	data := line.data

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
					Ch: invisibleRuneTable[r],
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

		if line == v.cursor.line {
			// special case, cursor line
			v.drawLine(line, v.topLineNum+y, coff, v.lineVoffset)
		} else {
			v.drawLine(line, v.topLineNum+y, coff, 0)
		}

		coff += v.uiBuf.Width
		line = line.next
	}
}

func (v *view) drawStatus() {
	if v.oneline {
		return
	}

	// fill background with '─'
	lp := tulib.DefaultLabelParams
	lp.Bg = termbox.AttrReverse
	lp.Fg = termbox.AttrReverse | termbox.AttrBold
	v.uiBuf.Fill(tulib.Rect{0, v.height(), v.uiBuf.Width, 1}, termbox.Cell{
		Fg: termbox.AttrReverse,
		Bg: termbox.AttrReverse,
		Ch: '─',
	})

	// on disk sync status
	if !v.buf.syncedWithDisk() {
		cell := termbox.Cell{
			Fg: termbox.AttrReverse,
			Bg: termbox.AttrReverse,
			Ch: '*',
		}
		v.uiBuf.Set(1, v.height(), cell)
		v.uiBuf.Set(2, v.height(), cell)
	}

	// filename
	fmt.Fprintf(&v.tmpBuf, "  %s  ", v.buf.name)
	v.uiBuf.DrawLabel(tulib.Rect{5, v.height(), v.uiBuf.Width, 1},
		&lp, v.tmpBuf.Bytes())
	namel := v.tmpBuf.Len()
	lp.Fg = termbox.AttrReverse
	v.tmpBuf.Reset()
	fmt.Fprintf(&v.tmpBuf, "(%d, %d)  ", v.cursor.lineNum, v.cursorVoffset)
	v.uiBuf.DrawLabel(tulib.Rect{5 + namel, v.height(), v.uiBuf.Width, 1},
		&lp, v.tmpBuf.Bytes())
	v.tmpBuf.Reset()
}

// Draw the current view to the 'v.uibuf'.
func (v *view) draw() {
	if v.dirty&DIRTY_CONTENTS != 0 {
		v.dirty &^= DIRTY_CONTENTS
		v.drawContents()
	}

	if v.dirty&DIRTY_STATUS != 0 {
		v.dirty &^= DIRTY_STATUS
		v.drawStatus()
	}
}

// Center view on the cursor.
func (v *view) centerViewOnCursor() {
	v.topLine = v.cursor.line
	v.topLineNum = v.cursor.lineNum
	v.moveTopLineNtimes(-v.height() / 2)
	v.dirty = DIRTY_EVERYTHING
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
	for top.prev != nil && n < 0 {
		top = top.prev
		v.topLineNum--
		n++
	}
	for top.next != nil && n > 0 {
		top = top.next
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

	cursor := v.cursor.line
	for cursor.prev != nil && n < 0 {
		cursor = cursor.prev
		v.cursor.lineNum--
		n++
	}
	for cursor.next != nil && n > 0 {
		cursor = cursor.next
		v.cursor.lineNum++
		n--
	}
	v.cursor.line = cursor
}

// When 'top_line' was changed, call this function to possibly adjust the
// 'cursor_line'.
func (v *view) adjustCursorLine() {
	vt := v.verticalThreshold()
	cursor := v.cursor.line
	co := v.cursor.lineNum - v.topLineNum
	h := v.height()

	if cursor.next != nil && co < vt {
		v.moveCursorLineNtimes(vt - co)
	}

	if cursor.prev != nil && co >= h-vt {
		v.moveCursorLineNtimes((h - vt) - co - 1)
	}

	if cursor != v.cursor.line {
		cursor = v.cursor.line
		bo, co, vo := cursor.findClosestOffsets(v.lastCursorVoffset)
		v.cursor.boffset = bo
		v.cursorCoffset = co
		v.cursorVoffset = vo
		v.lineVoffset = 0
		v.adjustLineVoffset()
		v.dirty = DIRTY_EVERYTHING
	}
}

// When 'cursor_line' was changed, call this function to possibly adjust the
// 'top_line'.
func (v *view) adjustTopLine() {
	vt := v.verticalThreshold()
	top := v.topLine
	co := v.cursor.lineNum - v.topLineNum
	h := v.height()

	if top.next != nil && co >= h-vt {
		v.moveTopLineNtimes(co - (h - vt) + 1)
		v.dirty = DIRTY_EVERYTHING
	}

	if top.prev != nil && co < vt {
		v.moveTopLineNtimes(co - vt)
		v.dirty = DIRTY_EVERYTHING
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
		v.dirty = DIRTY_EVERYTHING
	}
}

func (v *view) cursorPosition() (int, int) {
	y := v.cursor.lineNum - v.topLineNum
	x := v.cursorVoffset - v.lineVoffset
	return x, y
}

func (v *view) cursorPositionFor(cursor cursor) (int, int) {
	y := cursor.lineNum - v.topLineNum
	x := cursor.voffset() - v.lineVoffset
	return x, y
}

// Move cursor to the 'boffset' position in the 'line'. Obviously 'line' must be
// from the attached buffer. If 'boffset' < 0, use 'last_cursor_voffset'. Keep
// in mind that there is no need to maintain connections between lines (e.g. for
// moving from a deleted line to another line).
func (v *view) moveCursorTo(c cursor) {
	v.dirty |= DIRTY_STATUS
	if c.boffset < 0 {
		bo, co, vo := c.line.findClosestOffsets(v.lastCursorVoffset)
		v.cursor.boffset = bo
		v.cursorCoffset = co
		v.cursorVoffset = vo
	} else {
		vo, co := c.voffsetCoffset()
		v.cursor.boffset = c.boffset
		v.cursorCoffset = co
		v.cursorVoffset = vo
	}

	if c.boffset >= 0 {
		v.lastCursorVoffset = v.cursorVoffset
	}

	if c.line != v.cursor.line {
		if v.lineVoffset != 0 {
			v.dirty = DIRTY_EVERYTHING
		}
		v.lineVoffset = 0
	}
	v.cursor.line = c.line
	v.cursor.lineNum = c.lineNum
	v.adjustLineVoffset()
	v.adjustTopLine()
}

// Move cursor one character forward.
func (v *view) moveCursorForward() {
	c := v.cursor
	if c.lastLine() && c.eol() {
		v.ctx.setStatus("End of buffer")
		return
	}

	c.NextRune(configWrapRight)
	v.moveCursorTo(c)
}

// Move cursor one character backward.
func (v *view) moveCursorBackward() {
	c := v.cursor
	if c.firstLine() && c.bol() {
		v.ctx.setStatus("Beginning of buffer")
		return
	}

	c.PrevRune(configWrapLeft)
	v.moveCursorTo(c)
}

// Move cursor to the next line.
func (v *view) moveCursorNextLine() {
	c := v.cursor
	if !c.lastLine() {
		c = cursor{c.line.next, c.lineNum + 1, -1}
		v.moveCursorTo(c)
	} else {
		v.ctx.setStatus("End of buffer")
	}
}

// Move cursor to the previous line.
func (v *view) moveCursorPrevLine() {
	c := v.cursor
	if !c.firstLine() {
		c = cursor{c.line.prev, c.lineNum - 1, -1}
		v.moveCursorTo(c)
	} else {
		v.ctx.setStatus("Beginning of buffer")
	}
}

// Move cursor to the beginning of the line.
func (v *view) moveCursorBeginningOfLine() {
	c := v.cursor
	c.move_beginning_of_line()
	v.moveCursorTo(c)
}

// Move cursor to the end of the line.
func (v *view) moveCursorEndOfLine() {
	c := v.cursor
	c.move_end_of_line()
	v.moveCursorTo(c)
}

// Move cursor to the beginning of the file (buffer).
func (v *view) moveCursorBeginningOfFile() {
	c := cursor{v.buf.firstLine, 1, 0}
	v.moveCursorTo(c)
}

// Move cursor to the end of the file (buffer).
func (v *view) moveCursorEndOfFile() {
	c := cursor{v.buf.lastLine, v.buf.linesN, len(v.buf.lastLine.data)}
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
		v.dirty = DIRTY_EVERYTHING
	}
}

// Check if it's possible to move view 'n' lines forward or backward.
func (v *view) canMoveTopLineNtimes(n int) bool {
	if n == 0 {
		return true
	}

	top := v.topLine
	for top.prev != nil && n < 0 {
		top = top.prev
		n++
	}
	for top.next != nil && n > 0 {
		top = top.next
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

func (v *view) maybeNextActionGroup() {
	b := v.buf
	if b.history.next == nil {
		// no need to move
		return
	}

	prev := b.history
	b.history = b.history.next
	b.history.prev = prev
	b.history.next = nil
	b.history.actions = nil
	b.history.before = v.cursor
}

func (v *view) finalizeActionGroup() {
	b := v.buf
	// finalize only if we're at the tip of the undo history, this function
	// will be called mainly after each cursor movement and actions alike
	// (that are supposed to finalize action group)
	if b.history.next == nil {
		b.history.next = new(actionGroup)
		b.history.after = v.cursor
	}
}

func (v *view) undo() {
	b := v.buf
	if b.history.prev == nil {
		// we're at the sentinel, no more things to undo
		v.ctx.setStatus("No further undo information")
		return
	}

	// undo action causes finalization, always
	v.finalizeActionGroup()

	// undo invariant tells us 'len(b.history.actions) != 0' in case if this is
	// not a sentinel, revert the actions in the current action group
	for i := len(b.history.actions) - 1; i >= 0; i-- {
		a := &b.history.actions[i]
		a.revert(v)
	}
	v.moveCursorTo(b.history.before)
	v.lastCursorVoffset = v.cursorVoffset
	b.history = b.history.prev
	v.ctx.setStatus("Undo!")
}

func (v *view) redo() {
	b := v.buf
	if b.history.next == nil {
		// open group, obviously, can't move forward
		v.ctx.setStatus("No further redo information")
		return
	}
	if len(b.history.next.actions) == 0 {
		// last finalized group, moving to the next group breaks the
		// invariant and doesn't make sense (nothing to redo)
		v.ctx.setStatus("No further redo information")
		return
	}

	// move one entry forward, and redo all its actions
	b.history = b.history.next
	for i := range b.history.actions {
		a := &b.history.actions[i]
		a.apply(v)
	}
	v.moveCursorTo(b.history.after)
	v.lastCursorVoffset = v.cursorVoffset
	v.ctx.setStatus("Redo!")
}

func (v *view) actionInsert(c cursor, data []byte) {
	if v.oneline {
		data = bytes.Replace(data, []byte{'\n'}, nil, -1)
	}

	v.maybeNextActionGroup()
	a := action{
		what:   actionInsert,
		data:   data,
		cursor: c,
		lines:  make([]*line, bytes.Count(data, []byte{'\n'})),
	}
	for i := range a.lines {
		a.lines[i] = new(line)
	}
	a.apply(v)
	v.buf.history.append(&a)
}

func (v *view) actionDelete(c cursor, nbytes int) {
	v.maybeNextActionGroup()
	d := c.extractBytes(nbytes)
	a := action{
		what:   actionDelete,
		data:   d,
		cursor: c,
		lines:  make([]*line, bytes.Count(d, []byte{'\n'})),
	}
	for i := range a.lines {
		a.lines[i] = c.line.next
		c.line = c.line.next
	}
	a.apply(v)
	v.buf.history.append(&a)
}

// Insert a rune 'r' at the current cursor position, advance cursor one character forward.
func (v *view) insertRune(r rune) {
	var data [utf8.UTFMax]byte
	l := utf8.EncodeRune(data[:], r)
	c := v.cursor
	if r == '\n' || r == '\r' {
		v.actionInsert(c, []byte{'\n'})
		prev := c.line
		c.line = c.line.next
		c.lineNum++
		c.boffset = 0

		if r == '\n' {
			i := indexFirstNonSpace(prev.data)
			if i > 0 {
				autoindent := cloneByteSlice(prev.data[:i])
				v.actionInsert(c, autoindent)
				c.boffset += len(autoindent)
			}
		}
	} else {
		v.actionInsert(c, data[:l])
		c.boffset += l
	}
	v.moveCursorTo(c)
	v.dirty = DIRTY_EVERYTHING
}

// If at the beginning of the line, move contents of the current line to the end
// of the previous line. Otherwise, erase one character backward.
func (v *view) deleteRuneBackward() {
	c := v.cursor
	if c.bol() {
		if c.firstLine() {
			// beginning of the file
			v.ctx.setStatus("Beginning of buffer")
			return
		}
		c.line = c.line.prev
		c.lineNum--
		c.boffset = len(c.line.data)
		v.actionDelete(c, 1)
		v.moveCursorTo(c)
		v.dirty = DIRTY_EVERYTHING
		return
	}

	_, rlen := c.runeBefore()
	c.boffset -= rlen
	v.actionDelete(c, rlen)
	v.moveCursorTo(c)
	v.dirty = DIRTY_EVERYTHING
}

// If at the EOL, move contents of the next line to the end of the current line,
// erasing the next line after that. Otherwise, delete one character under the
// cursor.
func (v *view) deleteRune() {
	c := v.cursor
	if c.eol() {
		if c.lastLine() {
			// end of the file
			v.ctx.setStatus("End of buffer")
			return
		}
		v.actionDelete(c, 1)
		v.dirty = DIRTY_EVERYTHING
		return
	}

	_, rlen := c.runeUnder()
	v.actionDelete(c, rlen)
	v.dirty = DIRTY_EVERYTHING
}

// If not at the EOL, remove contents of the current line from the cursor to the
// end. Otherwise behave like 'delete'.
func (v *view) killLine() {
	c := v.cursor
	if !c.eol() {
		// kill data from the cursor to the EOL
		len := len(c.line.data) - c.boffset
		v.appendToKillBuffer(c, len)
		v.actionDelete(c, len)
		v.dirty = DIRTY_EVERYTHING
		return
	}
	v.appendToKillBuffer(c, 1)
	v.deleteRune()
}

func (v *view) killWord() {
	c1 := v.cursor
	c2 := c1
	c2.moveOneWordForward()
	d := c1.distance(c2)
	if d > 0 {
		v.appendToKillBuffer(c1, d)
		v.actionDelete(c1, d)
	}
}

func (v *view) killWordBackward() {
	c2 := v.cursor
	c1 := c2
	c1.move_one_word_backward()
	d := c1.distance(c2)
	if d > 0 {
		v.prependToKillBuffer(c1, d)
		v.actionDelete(c1, d)
		v.moveCursorTo(c1)
	}
}

func (v *view) killRegion() {
	if !v.buf.isMarkSet() {
		v.ctx.setStatus("The mark is not set now, so there is no region")
		return
	}

	c1 := v.cursor
	c2 := v.buf.mark
	d := c1.distance(c2)
	switch {
	case d == 0:
		return
	case d < 0:
		d = -d
		v.appendToKillBuffer(c2, d)
		v.actionDelete(c2, d)
		v.moveCursorTo(c2)
	default:
		v.appendToKillBuffer(c1, d)
		v.actionDelete(c1, d)
	}
}

func (v *view) setMark() {
	v.buf.mark = v.cursor
	v.ctx.setStatus("Mark set")
}

func (v *view) swapCursorAndMark() {
	if v.buf.isMarkSet() {
		m := v.buf.mark
		v.buf.mark = v.cursor
		v.moveCursorTo(m)
	}
}

func (v *view) onInsertAdjustTopLine(a *action) {
	if a.cursor.lineNum < v.topLineNum && len(a.lines) > 0 {
		// inserted one or more lines above the view
		v.topLineNum += len(a.lines)
		v.dirty |= DIRTY_STATUS
	}
}

func (v *view) onDeleteAdjustTopLine(a *action) {
	if a.cursor.lineNum < v.topLineNum {
		// deletion above the top line
		if len(a.lines) == 0 {
			return
		}

		topnum := v.topLineNum
		first, last := a.deletedLines()
		if first <= topnum && topnum <= last {
			// deleted the top line, adjust the pointers
			if a.cursor.line.next != nil {
				v.topLine = a.cursor.line.next
				v.topLineNum = a.cursor.lineNum + 1
			} else {
				v.topLine = a.cursor.line
				v.topLineNum = a.cursor.lineNum
			}
			v.dirty = DIRTY_EVERYTHING
		} else {
			// no need to worry
			v.topLineNum -= len(a.lines)
			v.dirty |= DIRTY_STATUS
		}
	}
}

func (v *view) onInsert(a *action) {
	v.onInsertAdjustTopLine(a)
	if v.topLineNum+v.height() <= a.cursor.lineNum {
		// inserted something below the view, don't care
		return
	}
	if a.cursor.lineNum < v.topLineNum {
		// inserted something above the top line
		if len(a.lines) > 0 {
			// inserted one or more lines, adjust line numbers
			v.cursor.lineNum += len(a.lines)
			v.dirty |= DIRTY_STATUS
		}
		return
	}
	c := v.cursor
	c.onInsertAdjust(a)
	v.moveCursorTo(c)
	v.lastCursorVoffset = v.cursorVoffset
	v.dirty = DIRTY_EVERYTHING
}

func (v *view) onDelete(a *action) {
	v.onDeleteAdjustTopLine(a)
	if v.topLineNum+v.height() <= a.cursor.lineNum {
		// deleted something below the view, don't care
		return
	}
	if a.cursor.lineNum < v.topLineNum {
		// deletion above the top line
		if len(a.lines) == 0 {
			return
		}

		_, last := a.deletedLines()
		if last < v.topLineNum {
			// no need to worry
			v.cursor.lineNum -= len(a.lines)
			v.dirty |= DIRTY_STATUS
			return
		}
	}
	c := v.cursor
	c.onDeleteAdjust(a)
	v.moveCursorTo(c)
	v.lastCursorVoffset = v.cursorVoffset
	v.dirty = DIRTY_EVERYTHING
}

func (v *view) onVcommand(c viewCommand) {
	lastClass := v.lastCommand.Cmd.class()
	if c.Cmd.class() != lastClass || lastClass == vCommandClassMisc {
		v.finalizeActionGroup()
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
		case vCommandSetMark:
			v.setMark()
		case vCommandSwapCursorAndMark:
			v.swapCursorAndMark()
		case vCommandInsertRune:
			v.insertRune(c.Rune)
		case vCommandYank:
			v.yank()
		case vCommandDeleteRuneBackward:
			v.deleteRuneBackward()
		case vCommandDeleteRune:
			v.deleteRune()
		case vCommandKillLine:
			v.killLine()
		case vCommandKillWord:
			v.killWord()
		case vCommandKillWordBackward:
			v.killWordBackward()
		case vCommandKillRegion:
			v.killRegion()
		case vCommandCopyRegion:
			v.copyRegion()
		case vCommandUndo:
			v.undo()
		case vCommandRedo:
			v.redo()
		case vCommandIndentRegion:
			v.indentRegion()
		case vCommandDeindentRegion:
			v.deindentRegion()
		case vCommandRegionToUpper:
			v.regionTo(bytes.ToUpper)
		case vCommandRegionToLower:
			v.regionTo(bytes.ToLower)
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

func (v *view) cleanupTrailingWhitespaces() {
	cursor := cursor{
		line:    v.buf.firstLine,
		lineNum: 1,
		boffset: 0,
	}

	for cursor.line != nil {
		len := len(cursor.line.data)
		i := indexLastNonSpace(cursor.line.data)
		if i == -1 && len > 0 {
			// the whole string is whitespace
			v.actionDelete(cursor, len)
		}
		if i != -1 && i != len-1 {
			// some whitespace at the end
			cursor.boffset = i + 1
			v.actionDelete(cursor, len-cursor.boffset)
		}
		cursor.line = cursor.line.next
		cursor.lineNum++
		cursor.boffset = 0
	}

	// adjust cursor after changes possibly
	cursor = v.cursor
	if cursor.boffset > len(cursor.line.data) {
		cursor.boffset = len(cursor.line.data)
		v.moveCursorTo(cursor)
	}
}

func (v *view) cleanupTrailingNewlines() {
	cursor := cursor{
		line:    v.buf.lastLine,
		lineNum: v.buf.linesN,
		boffset: 0,
	}

	for len(cursor.line.data) == 0 {
		prev := cursor.line.prev
		if prev == nil {
			// beginning of the file, stop
			break
		}

		if len(prev.data) > 0 {
			// previous line is not empty, leave one empty line at
			// the end (trailing EOL)
			break
		}

		// adjust view cursor just in case
		if v.cursor.lineNum == cursor.lineNum {
			v.moveCursorPrevLine()
		}

		cursor.line = prev
		cursor.lineNum--
		cursor.boffset = 0
		v.actionDelete(cursor, 1)
	}
}

func (v *view) ensureTrailingEOL() {
	cursor := cursor{
		line:    v.buf.lastLine,
		lineNum: v.buf.linesN,
		boffset: len(v.buf.lastLine.data),
	}
	if len(v.buf.lastLine.data) > 0 {
		v.actionInsert(cursor, []byte{'\n'})
	}
}

func (v *view) presave_cleanup(raw bool) {
	v.finalizeActionGroup()
	v.lastCommand = viewCommand{Cmd: vCommandNone}
	if !raw {
		v.cleanupTrailingWhitespaces()
		v.cleanupTrailingNewlines()
		v.ensureTrailingEOL()
		v.finalizeActionGroup()
	}
}

func (v *view) appendToKillBuffer(cursor cursor, nbytes int) {
	kb := *v.ctx.killBuffer

	switch v.lastCommand.Cmd {
	case vCommandKillWord, vCommandKillWordBackward, vCommandKillRegion, vCommandKillLine:
	default:
		kb = kb[:0]
	}

	kb = append(kb, cursor.extractBytes(nbytes)...)
	*v.ctx.killBuffer = kb
}

func (v *view) prependToKillBuffer(cursor cursor, nbytes int) {
	kb := *v.ctx.killBuffer

	switch v.lastCommand.Cmd {
	case vCommandKillWord, vCommandKillWordBackward, vCommandKillRegion, vCommandKillLine:
	default:
		kb = kb[:0]
	}

	kb = append(cursor.extractBytes(nbytes), kb...)
	*v.ctx.killBuffer = kb
}

func (v *view) yank() {
	buf := *v.ctx.killBuffer
	cursor := v.cursor

	if len(buf) == 0 {
		return
	}
	cbuf := cloneByteSlice(buf)
	v.actionInsert(cursor, cbuf)
	for len(buf) > 0 {
		_, rlen := utf8.DecodeRune(buf)
		buf = buf[rlen:]
		cursor.NextRune(true)
	}
	v.moveCursorTo(cursor)
}

// shameless copy & paste from kill_region
func (v *view) copyRegion() {
	if !v.buf.isMarkSet() {
		v.ctx.setStatus("The mark is not set now, so there is no region")
		return
	}

	c1 := v.cursor
	c2 := v.buf.mark
	d := c1.distance(c2)
	switch {
	case d == 0:
		return
	case d < 0:
		d = -d
		v.appendToKillBuffer(c2, d)
	default:
		v.appendToKillBuffer(c1, d)
	}
}

// assumes that filtered text has the same length
func (v *view) regionTo(filter func([]byte) []byte) {
	if !v.buf.isMarkSet() {
		v.ctx.setStatus("The mark is not set now, so there is no region")
		return
	}
	v.filterText(v.cursor, v.buf.mark, filter)
}

func (v *view) lineRegion() (beg, end cursor) {
	beg = v.cursor
	end = v.cursor
	if v.buf.isMarkSet() {
		end = v.buf.mark
	}

	if beg.lineNum > end.lineNum {
		beg, end = end, beg
	}
	beg.boffset = 0
	end.boffset = len(end.line.data)
	return
}

func (v *view) indentLine(line cursor) {
	line.boffset = 0
	v.actionInsert(line, []byte{'\t'})
	if v.cursor.line == line.line {
		cursor := v.cursor
		cursor.boffset += 1
		v.moveCursorTo(cursor)
	}
}

func (v *view) deindentLine(line cursor) {
	line.boffset = 0
	if r, _ := line.runeUnder(); r == '\t' {
		v.actionDelete(line, 1)
	}
	if v.cursor.line == line.line && v.cursor.boffset > 0 {
		cursor := v.cursor
		cursor.boffset -= 1
		v.moveCursorTo(cursor)
	}
}

func (v *view) indentRegion() {
	beg, end := v.lineRegion()
	for beg.line != end.line {
		v.indentLine(beg)
		beg.line = beg.line.next
		beg.lineNum++
	}
	v.indentLine(end)
}

func (v *view) deindentRegion() {
	beg, end := v.lineRegion()
	for beg.line != end.line {
		v.deindentLine(beg)
		beg.line = beg.line.next
		beg.lineNum++
	}
	v.deindentLine(end)
}

func (v *view) wordTo(filter func([]byte) []byte) {
	c1, c2 := v.cursor, v.cursor
	c2.moveOneWordForward()
	v.filterText(c1, c2, filter)
	c1.moveOneWordForward()
	v.moveCursorTo(c1)
}

// Filter _must_ return a new slice and shouldn't touch contents of the
// argument, perfect filter examples are: bytes.Title, bytes.ToUpper,
// bytes.ToLower
func (v *view) filterText(from, to cursor, filter func([]byte) []byte) {
	c1, c2 := swapCursorMaybe(from, to)
	d := c1.distance(c2)
	v.actionDelete(c1, d)
	data := filter(v.buf.history.lastAction().data)
	v.actionInsert(c1, data)
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
	vCommandSetMark
	vCommandSwapCursorAndMark
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
