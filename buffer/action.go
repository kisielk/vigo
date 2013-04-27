package buffer

import (
	"bytes"
	"github.com/kisielk/vigo/utils"
)

//----------------------------------------------------------------------------
// action
//
// A single entity of undo/redo history. All changes to contents of a buffer
// must be initiated by an action.
//----------------------------------------------------------------------------

type actionType int

const (
	actionInsert actionType = 1
	actionDelete actionType = -1
)

type Action struct {
	what   actionType
	data   []byte
	cursor Cursor
	lines  []*Line
}

func (a *Action) Apply(buf *Buffer) {
	a.do(buf, a.what)
}

func (a *Action) Revert(buf *Buffer) {
	a.do(buf, -a.what)
}

func (a *Action) insertLine(line, prev *Line, buf *Buffer) {
	bi := prev
	ai := prev.next

	// 'bi' is always a non-nil line
	bi.next = line
	line.prev = bi

	// 'ai' could be nil (means we're inserting a new last line)
	if ai == nil {
		buf.LastLine = line
	} else {
		ai.prev = line
	}
	line.next = ai
}

func (a *Action) deleteLine(line *Line, buf *Buffer) {
	bi := line.prev
	ai := line.next
	if ai != nil {
		ai.prev = bi
	} else {
		buf.LastLine = bi
	}
	if bi != nil {
		bi.next = ai
	} else {
		buf.FirstLine = ai
	}
	line.data = line.data[:0]
}

func (a *Action) insert(buf *Buffer) {
	var data_chunk []byte
	nline := 0
	offset := a.cursor.Boffset
	line := a.cursor.Line
	utils.IterLines(a.data, func(data []byte) {
		if data[0] == '\n' {
			buf.numBytes++
			buf.numLines++

			if offset < len(line.data) {
				// a case where we insert at the middle of the
				// line, need to save that chunk for later
				// insertion at the end of the operation
				data_chunk = line.data[offset:]
				line.data = line.data[:offset]
			}
			// insert a line
			a.insertLine(a.lines[nline], line, buf)
			line = a.lines[nline]
			nline++
			offset = 0
		} else {
			buf.numBytes += len(data)

			// insert a chunk of data
			line.data = utils.InsertBytes(line.data, offset, data)
			offset += len(data)
		}
	})
	if data_chunk != nil {
		line.data = append(line.data, data_chunk...)
	}
	buf.Emit(BufferEvent{BufferEventInsert, a})
}

func (a *Action) delete(buf *Buffer) {
	nline := 0
	offset := a.cursor.Boffset
	line := a.cursor.Line
	utils.IterLines(a.data, func(data []byte) {
		if data[0] == '\n' {
			buf.numBytes--
			buf.numLines--

			// append the contents of the deleted line the current line
			line.data = append(line.data, a.lines[nline].data...)
			// delete a line
			a.deleteLine(a.lines[nline], buf)
			nline++
		} else {
			buf.numBytes -= len(data)

			// delete a chunk of data
			copy(line.data[offset:], line.data[offset+len(data):])
			line.data = line.data[:len(line.data)-len(data)]
		}
	})
	buf.Emit(BufferEvent{BufferEventDelete, a})
}

func (a *Action) do(buf *Buffer, what actionType) {
	switch what {
	case actionInsert:
		a.insert(buf)
	case actionDelete:
		a.delete(buf)
	}
}

func (a *Action) lastLine() *Line {
	return a.lines[len(a.lines)-1]
}

func (a *Action) lastLineAffectionLen() int {
	i := bytes.LastIndex(a.data, []byte{'\n'})
	if i == -1 {
		return len(a.data)
	}

	return len(a.data) - i - 1
}

func (a *Action) firstLineAffectionLen() int {
	i := bytes.Index(a.data, []byte{'\n'})
	if i == -1 {
		return len(a.data)
	}

	return i
}

// returns the range of deleted lines, the first and the last one
func (a *Action) deletedLines() (int, int) {
	first := a.cursor.LineNum + 1
	last := first + len(a.lines) - 1
	return first, last
}

func (a *Action) tryMerge(b *Action) bool {
	if a.what != b.what {
		// can only merge actions of the same type
		return false
	}

	if a.cursor.LineNum != b.cursor.LineNum {
		return false
	}

	if a.cursor.Boffset == b.cursor.Boffset {
		pa, pb := a, b
		if a.what == actionInsert {
			// on insertion merge as 'ba', on deletion as 'ab'
			pa, pb = pb, pa
		}
		pa.data = append(pa.data, pb.data...)
		pa.lines = append(pa.lines, pb.lines...)
		*a = *pa
		return true
	}

	// different boffsets, try to restore the sequence
	pa, pb := a, b
	if pb.cursor.Boffset < pa.cursor.Boffset {
		pa, pb = pb, pa
	}
	if pa.cursor.Boffset+len(pa.data) == pb.cursor.Boffset {
		pa.data = append(pa.data, pb.data...)
		pa.lines = append(pa.lines, pb.lines...)
		*a = *pa
		return true
	}
	return false
}

//----------------------------------------------------------------------------
// action group
//----------------------------------------------------------------------------

type ActionGroup struct {
	actions []Action
	next    *ActionGroup
	prev    *ActionGroup
	before  Cursor
	after   Cursor
}

func (ag *ActionGroup) append(a *Action) {
	if len(ag.actions) != 0 {
		// Oh, we have something in the group already, let's try to
		// merge this action with the last one.
		last := &ag.actions[len(ag.actions)-1]
		if last.tryMerge(a) {
			return
		}
	}
	ag.actions = append(ag.actions, *a)
}

// Valid only as long as no new actions were added to the action group.
func (ag *ActionGroup) LastAction() *Action {
	if len(ag.actions) == 0 {
		return nil
	}
	return &ag.actions[len(ag.actions)-1]
}
