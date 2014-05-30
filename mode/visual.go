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
			viewTag.AdjustEndOffset(-count)
		} else {
			viewTag.AdjustStartOffset(-count)
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward}, count}
	case 'j':
		if c.LineNum >= endLine {
			viewTag.AdjustEndLine(count)
		} else {
			viewTag.AdjustStartLine(count)
		}

		if c.Boffset < startPos {
			viewTag.SetEndOffset(c.Boffset)
		} else {
			viewTag.SetStartOffset(endPos)
			viewTag.SetEndOffset(c.Boffset)
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
	case 'k':
		if c.LineNum <= startLine {
			viewTag.AdjustStartLine(-count)
		} else {
			viewTag.AdjustEndLine(-count)
		}

		if c.Boffset < startPos {
			viewTag.SetEndOffset(startPos)
			viewTag.SetStartOffset(c.Boffset)
		}

		v.Tags[0] = viewTag
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
	case 'l':
		if c.Boffset >= endPos && c.LineNum == endLine {
			viewTag.AdjustEndOffset(count)
		} else {
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
