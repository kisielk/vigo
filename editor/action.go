package editor

import (
	"bytes"
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

type action struct {
	what   actionType
	data   []byte
	cursor cursor
	lines  []*line
}

func (a *action) apply(v *view) {
	a.do(v, a.what)
}

func (a *action) revert(v *view) {
	a.do(v, -a.what)
}

func (a *action) insertLine(line, prev *line, v *view) {
	bi := prev
	ai := prev.next

	// 'bi' is always a non-nil line
	bi.next = line
	line.prev = bi

	// 'ai' could be nil (means we're inserting a new last line)
	if ai == nil {
		v.buf.lastLine = line
	} else {
		ai.prev = line
	}
	line.next = ai
}

func (a *action) deleteLine(line *line, v *view) {
	bi := line.prev
	ai := line.next
	if ai != nil {
		ai.prev = bi
	} else {
		v.buf.lastLine = bi
	}
	if bi != nil {
		bi.next = ai
	} else {
		v.buf.firstLine = ai
	}
	line.data = line.data[:0]
}

func (a *action) insert(v *view) {
	var data_chunk []byte
	nline := 0
	offset := a.cursor.boffset
	line := a.cursor.line
	iterLines(a.data, func(data []byte) {
		if data[0] == '\n' {
			v.buf.bytesN++
			v.buf.linesN++

			if offset < len(line.data) {
				// a case where we insert at the middle of the
				// line, need to save that chunk for later
				// insertion at the end of the operation
				data_chunk = line.data[offset:]
				line.data = line.data[:offset]
			}
			// insert a line
			a.insertLine(a.lines[nline], line, v)
			line = a.lines[nline]
			nline++
			offset = 0
		} else {
			v.buf.bytesN += len(data)

			// insert a chunk of data
			line.data = insertBytes(line.data, offset, data)
			offset += len(data)
		}
	})
	if data_chunk != nil {
		line.data = append(line.data, data_chunk...)
	}
}

func (a *action) delete(v *view) {
	nline := 0
	offset := a.cursor.boffset
	line := a.cursor.line
	iterLines(a.data, func(data []byte) {
		if data[0] == '\n' {
			v.buf.bytesN--
			v.buf.linesN--

			// append the contents of the deleted line the current line
			line.data = append(line.data, a.lines[nline].data...)
			// delete a line
			a.deleteLine(a.lines[nline], v)
			nline++
		} else {
			v.buf.bytesN -= len(data)

			// delete a chunk of data
			copy(line.data[offset:], line.data[offset+len(data):])
			line.data = line.data[:len(line.data)-len(data)]
		}
	})
}

func (a *action) do(v *view, what actionType) {
	switch what {
	case actionInsert:
		a.insert(v)
		v.onInsertAdjustTopLine(a)
		v.buf.otherViews(v, func(v *view) {
			v.onInsert(a)
		})
		if v.buf.isMarkSet() {
			v.buf.mark.onInsertAdjust(a)
		}
	case actionDelete:
		a.delete(v)
		v.onDeleteAdjustTopLine(a)
		v.buf.otherViews(v, func(v *view) {
			v.onDelete(a)
		})
		if v.buf.isMarkSet() {
			v.buf.mark.onDeleteAdjust(a)
		}
	}
	v.dirty = DIRTY_EVERYTHING

	// any change to the buffer causes words cache invalidation
	v.buf.wordsCacheValid = false
}

func (a *action) lastLine() *line {
	return a.lines[len(a.lines)-1]
}

func (a *action) lastLineAffectionLen() int {
	i := bytes.LastIndex(a.data, []byte{'\n'})
	if i == -1 {
		return len(a.data)
	}

	return len(a.data) - i - 1
}

func (a *action) firstLineAffectionLen() int {
	i := bytes.Index(a.data, []byte{'\n'})
	if i == -1 {
		return len(a.data)
	}

	return i
}

// returns the range of deleted lines, the first and the last one
func (a *action) deletedLines() (int, int) {
	first := a.cursor.lineNum + 1
	last := first + len(a.lines) - 1
	return first, last
}

func (a *action) tryMerge(b *action) bool {
	if a.what != b.what {
		// can only merge actions of the same type
		return false
	}

	if a.cursor.lineNum != b.cursor.lineNum {
		return false
	}

	if a.cursor.boffset == b.cursor.boffset {
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
	if pb.cursor.boffset < pa.cursor.boffset {
		pa, pb = pb, pa
	}
	if pa.cursor.boffset+len(pa.data) == pb.cursor.boffset {
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

type actionGroup struct {
	actions []action
	next    *actionGroup
	prev    *actionGroup
	before  cursor
	after   cursor
}

func (ag *actionGroup) append(a *action) {
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
func (ag *actionGroup) lastAction() *action {
	if len(ag.actions) == 0 {
		return nil
	}
	return &ag.actions[len(ag.actions)-1]
}
