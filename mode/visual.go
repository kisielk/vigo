package mode

import (
	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/view"
	"github.com/kisielk/vigo/utils"
	"github.com/nsf/termbox-go"
)

type visualMode struct {
	editor *editor.Editor
	count  string
}

func NewVisualMode(e *editor.Editor) *visualMode {
	m := visualMode{editor: e}
	v := m.editor.ActiveView()
	c := v.Cursor()
	m.editor.SetStatus("Visual")

	viewTag := view.NewViewTag(
		c.LineNum,
		c.Boffset,
		c.LineNum,
		c.Boffset,
		termbox.ColorDefault,
		termbox.ColorDefault|termbox.AttrReverse,
	)

	v.Tags = append(v.Tags, viewTag)

	return &m
}

func (m *visualMode) OnKey(ev *termbox.Event) {
	count := utils.ParseCount(m.count)
	if count == 0 {
		count = 1
	}
	g := m.editor
	v := g.ActiveView()
	c := v.Cursor()
	viewTag := v.Tags[0]

	startLine, startPos := viewTag.StartPos()
	endLine, endPos := viewTag.EndPos()

	switch ev.Key {
	case termbox.KeyEsc:
		m.editor.SetMode(NewNormalMode(m.editor))
	}

	switch ev.Ch {
	case 'h':
		if c.Boffset >= startPos || c.LineNum != startLine {
			// cursor is right of the start of the selection
			// or anywhere on the line other that the start line
			// move the end of the selection left
			viewTag.AdjustEndOffset(-count)
		} else {
			// cursor is anywhere on the start line
			// move the start of the selection left
			viewTag.AdjustStartOffset(-count)
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward}, count}
	case 'j':
		if c.LineNum >= endLine {
			// cursor is below the end of the selection
			// move the end of the selection down
			viewTag.AdjustEndLine(count)
		} else {
			// cursor is above the selection
			// move the start of the selection down
			viewTag.AdjustStartLine(count)
		}

		// cursor is one line above the selection and further
		// along the line than the start of the selection
		// flip the start and end offsets of the selection
		if c.Boffset > endPos && c.LineNum == endLine -1 {
			viewTag.FlipStartAndEndOffsets()
		}

		// cursor is on the same line as the entire selection, and left of the end of
		// the selection.
		// flip the offsets
		if c.LineNum == endLine && c.LineNum == startLine && c.Boffset < endPos {
			viewTag.FlipStartAndEndOffsets()
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
	case 'k':
		if c.LineNum <= startLine {
			// cursor is above the start of the selection
			// move the start of the selection up
			viewTag.AdjustStartLine(-count)
		} else {
			// cursor is below the start of the selection
			// move the end of the selection up
			viewTag.AdjustEndLine(-count)
		}

		// cursor if right of the start of the selection and above
		// flip the start and end offsets of the selection
		if c.Boffset > startPos && c.LineNum <= startLine {
			viewTag.FlipStartAndEndOffsets()
		}
		
		// cursor is left of the start of the selection
		// and one line below.
		// flip the offsets
		if c.Boffset < startPos && c.LineNum == startLine +1 {
			viewTag.FlipStartAndEndOffsets()
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
	case 'l':
		if c.Boffset >= endPos && c.LineNum == endLine {
			// cursor is right of the end of the selection and on the same line
			// move the end of the selection right
			viewTag.AdjustEndOffset(count)
		} else {
			// cursor is left of the selection on any line
			// move the start of the selection right
			viewTag.AdjustStartOffset(count)
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Forward}, count}
	}

	// FIXME: there must be a better way of doing this
	// trigger a re-draw
	g.Resize()

	m.count = ""
}

func (m *visualMode) Exit() {
	v := m.editor.ActiveView()
	v.Tags = v.Tags[:0]
}
