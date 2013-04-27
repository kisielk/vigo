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

type ActionType int

const (
	ActionInsert ActionType = 1
	ActionDelete ActionType = -1
)

type Action struct {
	What   ActionType
	Data   []byte
	Cursor Cursor
	Lines  []*Line
}

func (a *Action) Apply(buf *Buffer) {
	a.do(buf, a.What)
}

func (a *Action) Revert(buf *Buffer) {
	a.do(buf, -a.What)
}

func (a *Action) InsertLine(line, prev *Line, buf *Buffer) {
	bi := prev
	ai := prev.Next

	// 'bi' is always a non-nil line
	bi.Next = line
	line.Prev = bi

	// 'ai' could be nil (means we're inserting a new last line)
	if ai == nil {
		buf.LastLine = line
	} else {
		ai.Prev = line
	}
	line.Next = ai
}

func (a *Action) DeleteLine(line *Line, buf *Buffer) {
	bi := line.Prev
	ai := line.Next
	if ai != nil {
		ai.Prev = bi
	} else {
		buf.LastLine = bi
	}
	if bi != nil {
		bi.Next = ai
	} else {
		buf.FirstLine = ai
	}
	line.Data = line.Data[:0]
}

func (a *Action) insert(buf *Buffer) {
	var data_chunk []byte
	nline := 0
	offset := a.Cursor.Boffset
	line := a.Cursor.Line
	utils.IterLines(a.Data, func(data []byte) {
		if data[0] == '\n' {
			buf.numBytes++
			buf.numLines++

			if offset < len(line.Data) {
				// a case where we insert at the middle of the
				// line, need to save that chunk for later
				// insertion at the end of the operation
				data_chunk = line.Data[offset:]
				line.Data = line.Data[:offset]
			}
			// insert a line
			a.InsertLine(a.Lines[nline], line, buf)
			line = a.Lines[nline]
			nline++
			offset = 0
		} else {
			buf.numBytes += len(data)

			// insert a chunk of data
			line.Data = utils.InsertBytes(line.Data, offset, data)
			offset += len(data)
		}
	})
	if data_chunk != nil {
		line.Data = append(line.Data, data_chunk...)
	}
	buf.Emit(BufferEvent{BufferEventInsert, a})
}

func (a *Action) delete(buf *Buffer) {
	nline := 0
	offset := a.Cursor.Boffset
	line := a.Cursor.Line
	utils.IterLines(a.Data, func(data []byte) {
		if data[0] == '\n' {
			buf.numBytes--
			buf.numLines--

			// append the contents of the deleted line the current line
			line.Data = append(line.Data, a.Lines[nline].Data...)
			// delete a line
			a.DeleteLine(a.Lines[nline], buf)
			nline++
		} else {
			buf.numBytes -= len(data)

			// delete a chunk of data
			copy(line.Data[offset:], line.Data[offset+len(data):])
			line.Data = line.Data[:len(line.Data)-len(data)]
		}
	})
	buf.Emit(BufferEvent{BufferEventDelete, a})
}

func (a *Action) do(buf *Buffer, what ActionType) {
	switch what {
	case ActionInsert:
		a.insert(buf)
	case ActionDelete:
		a.delete(buf)
	}
}

func (a *Action) LastLine() *Line {
	return a.Lines[len(a.Lines)-1]
}

func (a *Action) lastLineAffectionLen() int {
	i := bytes.LastIndex(a.Data, []byte{'\n'})
	if i == -1 {
		return len(a.Data)
	}

	return len(a.Data) - i - 1
}

func (a *Action) firstLineAffectionLen() int {
	i := bytes.Index(a.Data, []byte{'\n'})
	if i == -1 {
		return len(a.Data)
	}

	return i
}

// returns the range of deleted lines, the first and the last one
func (a *Action) DeletedLines() (int, int) {
	first := a.Cursor.LineNum + 1
	last := first + len(a.Lines) - 1
	return first, last
}

func (a *Action) tryMerge(b *Action) bool {
	if a.What != b.What {
		// can only merge actions of the same type
		return false
	}

	if a.Cursor.LineNum != b.Cursor.LineNum {
		return false
	}

	if a.Cursor.Boffset == b.Cursor.Boffset {
		pa, pb := a, b
		if a.What == ActionInsert {
			// on insertion merge as 'ba', on deletion as 'ab'
			pa, pb = pb, pa
		}
		pa.Data = append(pa.Data, pb.Data...)
		pa.Lines = append(pa.Lines, pb.Lines...)
		*a = *pa
		return true
	}

	// different boffsets, try to restore the sequence
	pa, pb := a, b
	if pb.Cursor.Boffset < pa.Cursor.Boffset {
		pa, pb = pb, pa
	}
	if pa.Cursor.Boffset+len(pa.Data) == pb.Cursor.Boffset {
		pa.Data = append(pa.Data, pb.Data...)
		pa.Lines = append(pa.Lines, pb.Lines...)
		*a = *pa
		return true
	}
	return false
}

//----------------------------------------------------------------------------
// action group
//----------------------------------------------------------------------------

type ActionGroup struct {
	Actions []Action
	Next    *ActionGroup
	Prev    *ActionGroup
	Before  Cursor
	After   Cursor
}

func (ag *ActionGroup) Append(a *Action) {
	if len(ag.Actions) != 0 {
		// Oh, we have something in the group already, let's try to
		// merge this action with the last one.
		last := &ag.Actions[len(ag.Actions)-1]
		if last.tryMerge(a) {
			return
		}
	}
	ag.Actions = append(ag.Actions, *a)
}

// Valid only as long as no new actions were added to the action group.
func (ag *ActionGroup) LastAction() *Action {
	if len(ag.Actions) == 0 {
		return nil
	}
	return &ag.Actions[len(ag.Actions)-1]
}
