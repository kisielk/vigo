package buffer

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"unicode/utf8"
	"path/filepath"

	"github.com/kisielk/vigo/utils"
)

type Line struct {
	Data []byte
	Next *Line
	Prev *Line
}

// Len returns the length of the line in bytes
func (l Line) Len() int {
	return len(l.Data)
}

// Find a set of closest offsets for a given visual offset
func (l *Line) FindClosestOffsets(voffset int) (bo, co, vo int) {
	data := l.Data
	for len(data) > 0 {
		var vodif int
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		vodif = utils.RuneAdvanceLen(r, vo)
		if vo+vodif > voffset {
			return
		}

		bo += rlen
		co += 1
		vo += vodif
	}
	return
}

type BufferEventType int

const (
	BufferEventInsert BufferEventType = iota
	BufferEventDelete
	BufferEventBOF
	BufferEventEOF
	BufferEventHistoryBack
	BufferEventHistoryStart
	BufferEventHistoryForward
	BufferEventHistoryEnd
	BufferEventSave
)

type BufferEvent struct {
	Type   BufferEventType
	Action *Action
}

type Buffer struct {
	FirstLine *Line
	LastLine  *Line
	NumLines  int
	numBytes  int
	History   *ActionGroup
	onDisk    *ActionGroup

	// absoulte path of the file, if it's empty string, then the file has no
	// on-disk representation
	Path string

	// buffer name (displayed in the status line), must be unique,
	// uniqueness is maintained by godit methods
	Name string

	listeners []chan BufferEvent
}

func NewEmptyBuffer() *Buffer {
	b := new(Buffer)
	l := new(Line)
	l.Next = nil
	l.Prev = nil
	b.FirstLine = l
	b.LastLine = l
	b.NumLines = 1
	b.listeners = []chan BufferEvent{}
	b.initHistory()
	return b
}

func (b *Buffer) AddListener(c chan BufferEvent) {
	b.listeners = append(b.listeners, c)
}

func (b *Buffer) RemoveListener(c chan BufferEvent) {
	for i := 0; i < len(b.listeners); i++ {
		if b.listeners[i] == c {
			b.listeners = append(b.listeners[:i], b.listeners[i+1:]...)
			return
		}
	}
}

func (b *Buffer) Emit(e BufferEvent) {
	for i := 0; i < len(b.listeners); i++ {
		b.listeners[i] <- e
	}
}

func NewBuffer(r io.Reader) (*Buffer, error) {
	var err error
	var prevline *Line

	br := bufio.NewReader(r)
	l := new(Line)
	b := new(Buffer)
	b.NumLines = 1
	b.FirstLine = l
	for {
		l.Data, err = br.ReadBytes('\n')
		if err != nil {
			// last line was read
			break
		} else {
			b.numBytes += len(l.Data)

			// cut off the '\n' character
			l.Data = l.Data[:len(l.Data)-1]
		}

		b.NumLines++
		l.Next = new(Line)
		l.Prev = prevline
		prevline = l
		l = l.Next
	}
	l.Prev = prevline
	b.LastLine = l

	// io.EOF is not an error
	if err == io.EOF {
		err = nil
	}

	// history
	b.initHistory()
	return b, err
}

// InsertRune inserts 'r' at the cursor position 'c'
func (b *Buffer) InsertRune(c Cursor, r rune) {
	if r == '\n' || r == '\r' {
		b.Insert(c, []byte{'\n'})
		prev := c.Line
		c.Line = c.Line.Next
		c.LineNum++
		c.Boffset = 0

		if r == '\n' {
			i := utils.IndexFirstNonSpace(prev.Data)
			if i > 0 {
				autoindent := utils.CloneByteSlice(prev.Data[:i])
				b.Insert(c, autoindent)
				c.Boffset += len(autoindent)
			}
		}
	} else {
		var data [utf8.UTFMax]byte
		nBytes := utf8.EncodeRune(data[:], r)
		b.Insert(c, data[:nBytes])
		c.Boffset += nBytes
	}
}

// If at the EOL, move contents of the next line to the end of the current line,
// erasing the next line after that. Otherwise, delete one character under the
// cursor.
func (b *Buffer) DeleteRune(c Cursor) {
	if c.EOL() {
		if c.LastLine() {
			// end of the file
			b.Emit(BufferEvent{Type: BufferEventEOF})
			return
		}
		b.Delete(c, 1)
		return
	}

	_, rlen := c.RuneUnder()
	b.Delete(c, rlen)
}

// If at the beginning of the line, move contents of the current line to the end
// of the previous line. Otherwise, erase one character backward.
func (b *Buffer) DeleteRuneBackward(c Cursor) {
	if c.BOL() {
		if c.FirstLine() {
			// beginning of the file
			b.Emit(BufferEvent{Type: BufferEventBOF})
			return
		}
		c.Line = c.Line.Prev
		c.LineNum--
		c.Boffset = c.Line.Len()
		b.Delete(c, 1)
		return
	}

	_, rlen := c.RuneBefore()
	c.Boffset -= rlen
	b.Delete(c, rlen)
}

// InsertLine inserts a line after prev in the buffer.
// If prev is nil then the line will be the new first line of the buffer.
func (b *Buffer) InsertLine(line *Line, prev *Line) {
	// NOTE: 1) does not update b.numBytes
	bi := prev
	ai := b.FirstLine
	if bi != nil {
		ai = prev.Next
		bi.Next = line
		line.Prev = bi
	} else {
		b.FirstLine = line
	}

	// 'ai' could be nil (means we're inserting a new last line)
	if ai == nil {
		b.LastLine = line
	} else {
		ai.Prev = line
	}

	line.Next = ai
	b.NumLines++
}

func (b *Buffer) DeleteLine(line *Line) {
	// NOTE: 1) does not update b.numBytes
	//       2) zeroes line bytes
	bi := line.Prev
	ai := line.Next
	if ai != nil {
		ai.Prev = bi
	} else {
		b.LastLine = bi
	}
	if bi != nil {
		bi.Next = ai
	} else {
		b.FirstLine = ai
	}
	line.Data = line.Data[:0]
	b.NumLines--
}

// maybeNextActionGroup moves history forward one action group and
// discards any further redo action groups. This is done when
// the buffer is modified after several undo's.
func (b *Buffer) maybeNextActionGroup() {
	if b.History.Next == nil {
		// no need to move
		return
	}

	prev := b.History
	b.History = b.History.Next
	b.History.Prev = prev
	b.History.Next = nil
	b.History.Actions = nil
}

func (b *Buffer) FinalizeActionGroup() {
	// finalize only if we're at the tip of the undo history, this function
	// will be called mainly after each cursor movement and actions alike
	// (that are supposed to finalize action group)
	if b.History.Next == nil {
		b.History.Next = new(ActionGroup)
	}
}

func (b *Buffer) Insert(c Cursor, data []byte) {
	b.maybeNextActionGroup()

	a := NewInsertAction(c, data)
	a.Apply(b)

	b.History.Append(a)
}

func (b *Buffer) Delete(c Cursor, numBytes int) {
	b.maybeNextActionGroup()

	a := NewDeleteAction(c, numBytes)
	a.Apply(b)

	b.History.Append(a)
}

func (b *Buffer) DeleteRange(from Cursor, to Cursor) {
	b.Delete(from, from.Distance(to))
}

func (b *Buffer) Undo() {
	if b.History.Prev == nil {
		// we're at the sentinel, no more things to undo
		b.Emit(BufferEvent{Type: BufferEventHistoryStart})
		return
	}
	// undo action causes finalization, always
	b.FinalizeActionGroup()
	// undo invariant tells us 'len(b.history.actions) != 0' in case if this is
	// not a sentinel, revert the actions in the current action group
	for i := len(b.History.Actions) - 1; i >= 0; i-- {
		a := &b.History.Actions[i]
		a.Revert(b)
	}
	b.History = b.History.Prev
	b.Emit(BufferEvent{Type: BufferEventHistoryBack})
}

func (b *Buffer) Redo() {
	if b.History.Next == nil {
		// open group, obviously, can't move forward
		b.Emit(BufferEvent{Type: BufferEventHistoryEnd})
		return
	}
	if len(b.History.Next.Actions) == 0 {
		// last finalized group, moving to the next group breaks the
		// invariant and doesn't make sense (nothing to redo)
		b.Emit(BufferEvent{Type: BufferEventHistoryEnd})
		return
	}
	// move one entry forward, and redo all its actions
	b.History = b.History.Next
	for i := range b.History.Actions {
		a := &b.History.Actions[i]
		a.Apply(b)
	}
	b.Emit(BufferEvent{Type: BufferEventHistoryForward})
}

// CleanupTrailingSpaces removes trailing whitespace
// characters from every line in the buffer.
func (b *Buffer) CleanupTrailingSpaces() {
	cursor := Cursor{
		Line:    b.FirstLine,
		LineNum: 1,
		Boffset: 0,
	}
	for cursor.Line != nil {
		llen := cursor.Line.Len()
		i := utils.IndexLastNonSpace(cursor.Line.Data)
		if i == -1 && llen > 0 {
			// the whole string is whitespace
			b.Delete(cursor, llen)
		}
		if i != -1 && i != llen-1 {
			// some whitespace at the end
			cursor.Boffset = i + 1
			b.Delete(cursor, llen-cursor.Boffset)
		}
		cursor.Line = cursor.Line.Next
		cursor.LineNum++
		cursor.Boffset = 0
	}
}

// CleanupTrailingNewlines removes all but one trailing newlines.
func (b *Buffer) CleanupTrailingNewlines() {
	line := b.LastLine
	for line.Len() == 0 {
		prev := line.Prev
		if prev == nil || prev.Len() > 0 {
			// Either beginning of the file or previous line is
			// not empty; leave one empty line at the end
			break
		}
		b.DeleteLine(line)
		line = prev
	}
}

// EnsureTrailingEOL adds a newline at the end of the buffer, unless one exists already.
func (b *Buffer) EnsureTrailingEOL() {
	if b.LastLine.Len() == 0 {
		return
	}
	c := Cursor{
		Line:    b.LastLine,
		LineNum: b.NumLines,
		Boffset: b.LastLine.Len(),
	}
	b.Insert(c, []byte{'\n'})
}

func (b *Buffer) initHistory() {
	// the trick here is that I set 'sentinel' as 'history', it is required
	// to maintain an invariant, where 'history' is a sentinel or is not
	// empty

	sentinel := new(ActionGroup)
	first := new(ActionGroup)
	sentinel.Next = first
	first.Prev = sentinel
	b.History = sentinel
	b.onDisk = sentinel
}

func (b *Buffer) dumpHistory() {
	cur := b.History
	for cur.Prev != nil {
		cur = cur.Prev
	}

	p := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}

	i := 0
	for cur != nil {
		p("action group %d: %d actions\n", i, len(cur.Actions))
		for _, a := range cur.Actions {
			switch a.What {
			case ActionInsert:
				p(" + insert")
			case ActionDelete:
				p(" - delete")
			}
			p(" (%2d,%2d):%q\n", a.Cursor.LineNum,
				a.Cursor.Boffset, string(a.Data))
		}
		cur = cur.Next
		i++
	}
}

func (b *Buffer) Save() error {
	return b.SaveAs(b.Path)
}

func (b *Buffer) SaveAs(filename string) error {

	// TODO configure cleanup
	b.CleanupTrailingSpaces()
	b.CleanupTrailingNewlines()
	b.EnsureTrailingEOL()

	fullpath, err := filepath.Abs(filename)
	if err != nil {
		return error( fmt.Errorf("couldn't determine absolute path: %s", err))
	}

	r := b.reader()
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}

	b.onDisk = b.History
	if b.Path != ""{
		b.Path = fullpath
	}
	b.Emit(BufferEvent{Type: BufferEventSave})

	return nil
}

func (b *Buffer) SyncedWithDisk() bool {
	return b.onDisk == b.History
}

func (b *Buffer) reader() *BufferReader {
	return newBufferReader(b)
}

func (b *Buffer) contents() []byte {
	data, _ := ioutil.ReadAll(b.reader())
	return data
}

type BufferReader struct {
	buffer *Buffer
	Line   *Line
	offset int
}

func newBufferReader(buffer *Buffer) *BufferReader {
	br := new(BufferReader)
	br.buffer = buffer
	br.Line = buffer.FirstLine
	br.offset = 0
	return br
}

func (br *BufferReader) Read(data []byte) (int, error) {
	nread := 0
	for len(data) > 0 {
		if br.Line == nil {
			return nread, io.EOF
		}

		// how much can we read from current line
		canRead := len(br.Line.Data) - br.offset
		if len(data) <= canRead {
			// if this is all we need, return
			n := copy(data, br.Line.Data[br.offset:])
			nread += n
			br.offset += n
			break
		}

		// otherwise try to read '\n' and jump to the next line
		n := copy(data, br.Line.Data[br.offset:])
		nread += n
		data = data[n:]
		if len(data) > 0 && br.Line != br.buffer.LastLine {
			data[0] = '\n'
			data = data[1:]
			nread++
		}

		br.Line = br.Line.Next
		br.offset = 0
	}
	return nread, nil
}
