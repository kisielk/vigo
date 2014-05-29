package view

import (
	"bytes"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/kisielk/vigo/buffer"
	"github.com/kisielk/vigo/utils"
	"github.com/nsf/termbox-go"
	"github.com/nsf/tulib"
)

type dirtyFlag int

const (
	VerticalThreshold   = 5
	HorizontalThreshold = 10
)

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

func (l viewLocation) Cursor() buffer.Cursor {
	return l.cursor
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

type Context struct {
	setStatus  StatusFunc
	killBuffer *[]byte
	buffers    *[]*buffer.Buffer
}

type StatusFunc func(format string, args ...interface{})

func NewContext(setStatus StatusFunc, killBuffer *[]byte, buffers *[]*buffer.Buffer) Context {
	return Context{setStatus, killBuffer, buffers}
}

//----------------------------------------------------------------------------
// view
//
// Think of it as a window. It draws contents from a portion of a buffer into
// 'uibuf' and maintains things like cursor position.
//----------------------------------------------------------------------------

type View struct {
	viewLocation
	ctx             Context
	buf             *buffer.Buffer // currently displayed buffer
	uiBuf           tulib.Buffer
	dirty           dirtyFlag
	highlightBytes  []byte
	highlightRanges []byteRange
	tags            []viewTag
	redraw          chan struct{}

	// statusBuf is a buffer used for drawing the status line
	statusBuf bytes.Buffer

	bufferEvents chan buffer.BufferEvent
}

// SetStatus sets the status line of the view
func (v *View) SetStatus(format string, args ...interface{}) {
	v.ctx.setStatus(format, args...)
}

func (v *View) Buffer() *buffer.Buffer {
	return v.buf
}

func NewView(ctx Context, buf *buffer.Buffer, redraw chan struct{}) *View {
	v := &View{
		ctx:             ctx,
		uiBuf:           tulib.NewBuffer(1, 1),
		highlightRanges: make([]byteRange, 0, 10),
		tags:            make([]viewTag, 0, 10),
		redraw:          redraw,
	}
	v.Attach(buf)
	return v
}

func (v *View) UIBuf() tulib.Buffer {
	return v.uiBuf
}

func (v *View) Attach(b *buffer.Buffer) {
	if v.buf == b {
		return
	}
	if v.buf != nil {
		v.Detach()
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
	v.buf.AddListener(v.bufferEvents)
	go v.bufferEventLoop()

	v.dirty = dirtyEverything
}

func (v *View) bufferEventLoop() {
	for e := range v.bufferEvents {
		switch e.Type {
		case buffer.BufferEventInsert:
			v.onInsertAdjustTopLine(e.Action)
			c := v.cursor
			c.OnInsertAdjust(e.Action)
			v.MoveCursorTo(c)
			v.dirty = dirtyEverything
			// FIXME for unfocused views, just call onInsert
			// v.onInsert(e.Action)
		case buffer.BufferEventDelete:
			v.onDeleteAdjustTopLine(e.Action)
			c := v.cursor
			c.OnDeleteAdjust(e.Action)
			v.MoveCursorTo(c)
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
}

func (v *View) Detach() {
	// Stop listening to current buffer and close event loop.
	v.buf.RemoveListener(v.bufferEvents)
	close(v.bufferEvents)
	v.bufferEvents = nil
	v.buf = nil
}

// Resize the 'v.uibuf', adjusting things accordingly.
func (v *View) resize(w, h int) {
	v.uiBuf.Resize(w, h)
	v.adjustLineVoffset()
	v.adjustTopLine()
	v.dirty = dirtyEverything
}

func (v *View) height() int {
	return v.uiBuf.Height - 1
}

func (v *View) verticalThreshold() int {
	maxVthreshold := (v.height() - 1) / 2
	if VerticalThreshold > maxVthreshold {
		return maxVthreshold
	}
	return VerticalThreshold
}

func (v *View) horizontalThreshold() int {
	max_h_threshold := (v.width() - 1) / 2
	if HorizontalThreshold > max_h_threshold {
		return max_h_threshold
	}
	return HorizontalThreshold
}

func (v *View) width() int {
	return v.uiBuf.Width
}

func (v *View) drawLine(line *buffer.Line, lineNum, coff, lineVoffset int) {
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
			tabstop += utils.TabstopLength
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

func (v *View) drawContents() {
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

func (v *View) drawStatus() {
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
func (v *View) draw() {
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
func (v *View) centerViewOnCursor() {
	v.topLine = v.cursor.Line
	v.topLineNum = v.cursor.LineNum
	v.moveTopLineNtimes(-v.height() / 2)
	v.dirty = dirtyEverything
}

func (v *View) MoveCursorToLine(n int) {
	v.moveCursorBeginningOfFile()
	v.moveCursorLineNtimes(n - 1)
	v.centerViewOnCursor()
}

// Move top line 'n' times forward or backward.
func (v *View) moveTopLineNtimes(n int) {
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
func (v *View) moveCursorLineNtimes(n int) {
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
func (v *View) adjustCursorLine() {
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
func (v *View) adjustTopLine() {
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
func (v *View) adjustLineVoffset() {
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

func (v *View) CursorPosition() (int, int) {
	y := v.cursor.LineNum - v.topLineNum
	x := v.cursorVoffset - v.lineVoffset
	return x, y
}

// Move cursor to the 'boffset' position in the 'line'. Obviously 'line' must be
// from the attached buffer. If 'boffset' < 0, use 'last_cursor_voffset'. Keep
// in mind that there is no need to maintain connections between lines (e.g. for
// moving from a deleted line to another line).
func (v *View) MoveCursorTo(c buffer.Cursor) {
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

// Move cursor to the beginning of the file (buffer).
func (v *View) moveCursorBeginningOfFile() {
	c := buffer.Cursor{
		Line:    v.buf.FirstLine,
		LineNum: 1,
		Boffset: 0,
	}
	v.MoveCursorTo(c)
}

// Move view 'n' lines forward or backward.
func (v *View) MoveViewLines(n int) {
	prevtop := v.topLineNum
	v.moveTopLineNtimes(n)
	if prevtop != v.topLineNum {
		v.adjustCursorLine()
		v.dirty = dirtyEverything
	}
}

// Check if it's possible to move view 'n' lines forward or backward.
func (v *View) canMoveTopLineNtimes(n int) bool {
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
func (v *View) maybeMoveViewNlines(n int) {
	if v.canMoveTopLineNtimes(n) {
		v.MoveViewLines(n)
	}
}

func (v *View) onInsertAdjustTopLine(a *buffer.Action) {
	if a.Cursor.LineNum < v.topLineNum && len(a.Lines) > 0 {
		// inserted one or more lines above the view
		v.topLineNum += len(a.Lines)
		v.dirty |= dirtyStatus
	}
}

func (v *View) onDeleteAdjustTopLine(a *buffer.Action) {
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

func (v *View) onInsert(a *buffer.Action) {
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
	v.MoveCursorTo(c)
	v.lastCursorVoffset = v.cursorVoffset
	v.dirty = dirtyEverything
}

func (v *View) onDelete(a *buffer.Action) {
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
	v.MoveCursorTo(c)
	v.lastCursorVoffset = v.cursorVoffset
	v.dirty = dirtyEverything
}

func (v *View) dumpInfo() {
	p := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}

	p("Top line num: %d\n", v.topLineNum)
}

func (v *View) findHighlightRangesForLine(data []byte) {
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

func (v *View) inOneOfHighlightRanges(offset int) bool {
	for _, r := range v.highlightRanges {
		if r.includes(offset) {
			return true
		}
	}
	return false
}

func (v *View) tag(line, offset int) *viewTag {
	for i := range v.tags {
		t := &v.tags[i]
		if t.includes(line, offset) {
			return t
		}
	}
	return &defaultViewTag
}

func (v *View) makeCell(line, offset int, ch rune) termbox.Cell {
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

func (v *View) yank() {
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
	v.MoveCursorTo(cursor)
}

func (v *View) indentLine(line buffer.Cursor) {
	line.Boffset = 0
	v.buf.Insert(line, []byte{'\t'})
	if v.cursor.Line == line.Line {
		cursor := v.cursor
		cursor.Boffset += 1
		v.MoveCursorTo(cursor)
	}
}

func (v *View) deindentLine(line buffer.Cursor) {
	line.Boffset = 0
	if r, _ := line.RuneUnder(); r == '\t' {
		v.buf.Delete(line, 1)
	}
	if v.cursor.Line == line.Line && v.cursor.Boffset > 0 {
		cursor := v.cursor
		cursor.Boffset -= 1
		v.MoveCursorTo(cursor)
	}
}

// Filter _must_ return a new slice and shouldn't touch contents of the
// argument, perfect filter examples are: bytes.Title, bytes.ToUpper,
// bytes.ToLower
func (v *View) filterText(from, to buffer.Cursor, filter func([]byte) []byte) {
	c1, c2 := buffer.SortCursors(from, to)
	d := c1.Distance(c2)
	v.buf.Delete(c1, d)
	data := filter(v.buf.History.LastAction().Data)
	v.buf.Insert(c1, data)
}
