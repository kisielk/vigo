package commands

import (
	"github.com/kisielk/vigo/editor"
)

type AdjustSelection struct {
	Dir      Dir
	LineMode bool
}

func (cmd AdjustSelection) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()

	vRange := v.VisualRange()
	startLine, startPos := vRange.StartPos()
	endLine, endPos := vRange.EndPos()

	switch cmd.Dir {
	case Forward:
		if !cmd.LineMode {
			if c.Boffset >= endPos && c.LineNum == endLine {
				// cursor is right of the end of the selection and on the same line
				// move the end of the selection right
				vRange.AdjustEndOffset(1)
			} else {
				// cursor is left of the selection on any line
				// move the start of the selection right
				vRange.AdjustStartOffset(1)
			}

			v.SetVisualRange(vRange)
		}
	case Backward:
		if !cmd.LineMode {
			if c.Boffset >= startPos || c.LineNum != startLine {
				// cursor is right of the start of the selection
				// or anywhere on the line other that the start line
				// move the end of the selection left
				vRange.AdjustEndOffset(-1)
			} else {
				// cursor is anywhere on the start line
				// move the start of the selection left
				vRange.AdjustStartOffset(-1)
			}

			v.SetVisualRange(vRange)
		}
	case Up:
		if c.LineNum <= startLine {
			// cursor is above the start of the selection
			// move the start of the selection up
			vRange.AdjustStartLine(-1)
		} else {
			// cursor is below the start of the selection
			// move the end of the selection up
			vRange.AdjustEndLine(-1)
		}

		// cursor if right of the start of the selection and above
		// flip the start and end offsets of the selection
		if c.Boffset > startPos && c.LineNum <= startLine {
			vRange.FlipStartAndEndOffsets()
		}

		// cursor is left of the start of the selection
		// and one line below.
		// flip the offsets
		if c.Boffset < startPos && c.LineNum == startLine+1 {
			vRange.FlipStartAndEndOffsets()
		}

		v.SetVisualRange(vRange)
	case Down:
		if c.LineNum >= endLine {
			// cursor is below the end of the selection
			// move the end of the selection down
			vRange.AdjustEndLine(1)
		} else {
			// cursor is above the selection
			// move the start of the selection down
			vRange.AdjustStartLine(1)
		}

		// cursor is one line above the selection and further
		// along the line than the start of the selection
		// flip the start and end offsets of the selection
		if c.Boffset > endPos && c.LineNum == endLine-1 {
			vRange.FlipStartAndEndOffsets()
		}

		// cursor is on the same line as the entire selection, and left of the end of
		// the selection.
		// flip the offsets
		if c.LineNum == endLine && c.LineNum == startLine && c.Boffset < endPos {
			vRange.FlipStartAndEndOffsets()
		}

		v.SetVisualRange(vRange)
	}
}
