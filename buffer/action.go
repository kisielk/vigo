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

// NewInsertAction creates a new action inserting data bytes at c.
func NewInsertAction(c Cursor, data []byte) *Action {
	a := Action{
		What:   ActionInsert,
		Data:   data,
		Cursor: c,
		Lines:  make([]*Line, bytes.Count(data, []byte{'\n'})),
	}
	for i := range a.Lines {
		a.Lines[i] = new(Line)
	}
	return &a
}

// NewDeleteAction creates a new action deleting numBytes bytes at c.
func NewDeleteAction(c Cursor, numBytes int) *Action {
	d := c.ExtractBytes(numBytes)
	a := Action{
		What:   ActionDelete,
		Data:   d,
		Cursor: c,
		Lines:  make([]*Line, bytes.Count(d, []byte{'\n'})),
	}
	for i := range a.Lines {
		a.Lines[i] = c.Line.Next
		c.Line = c.Line.Next
	}
	return &a
}

func (a *Action) Apply(buf *Buffer) {
	a.do(buf, a.What)
}

func (a *Action) Revert(buf *Buffer) {
	a.do(buf, -a.What)
}

func (a *Action) insert(buf *Buffer) {
	var data_chunk []byte
	nline := 0
	offset := a.Cursor.Boffset
	line := a.Cursor.Line
	utils.IterLines(a.Data, func(data []byte) {
		if data[0] == '\n' {
			buf.numBytes++
			if offset < len(line.Data) {
				// a case where we insert at the middle of the
				// line, need to save that chunk for later
				// insertion at the end of the operation
				data_chunk = line.Data[offset:]
				line.Data = line.Data[:offset]
			}
			// insert a line
			buf.InsertLine(a.Lines[nline], line)
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
			// append the contents of the deleted line the current line
			line.Data = append(line.Data, a.Lines[nline].Data...)
			// delete a line
			buf.DeleteLine(a.Lines[nline])
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

// CursorBefore returns cursor position before actions in the group are applied.
func (ag *ActionGroup) CursorBefore() Cursor {
	// FIXME for now, return the cursor of the first action.
	// This is not accurate for cases like merged deletions, where
	// we need to return cursor + deletion length.
	if len(ag.Actions) == 0 {
		// TODO return some sentinel cursor value instead?
		panic("action group is empty")
	}
	return ag.Actions[0].Cursor
}

// CursorBefore returns cursor position after actions in the group are applied.
func (ag *ActionGroup) CursorAfter() Cursor {
	// FIXME this is inaccurate for same reasons as CursorBefore()
	return ag.Actions[len(ag.Actions)-1].Cursor
}
