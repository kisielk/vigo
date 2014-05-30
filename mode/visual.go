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
	lineMode bool
}

func NewVisualMode(e *editor.Editor, lineMode bool) *visualMode {
	m := visualMode{editor: e, lineMode: lineMode}
	v := m.editor.ActiveView()
	c := v.Cursor()

	startPos, endPos := 0, 0

	if lineMode {
		m.editor.SetStatus("Visual Line")
	} else {
		startPos = c.Boffset
		endPos = c.Boffset

		m.editor.SetStatus("Visual")
	}

	viewTag := view.NewTag(
		c.LineNum,
		startPos,
		c.LineNum,
		endPos,
		termbox.ColorDefault,
		termbox.ColorDefault|termbox.AttrReverse,
	)

	v.SetVisualRange(&viewTag)

	return &m
}

func (m *visualMode) OnKey(ev *termbox.Event) {

	// Consequtive non-zero digits specify action multiplier;
	// accumulate and return. Accept zero only if it's
	// a non-starting character.
	if ('0' < ev.Ch && ev.Ch <= '9') || (ev.Ch == '0' && len(m.count) > 0) {
		m.count = m.count + string(ev.Ch)
		m.editor.SetStatus(m.count)
		return
	}
	count := utils.ParseCount(m.count)
	if count == 0 {
		count = 1
	}
	g := m.editor
	v := g.ActiveView()
	c := v.Cursor()

	switch ev.Key {
	case termbox.KeyEsc:
		m.editor.SetMode(NewNormalMode(m.editor))
	}

	switch ev.Ch {
	case 'h':
		g.Commands <- cmd.Repeat{cmd.AdjustSelection{Dir: cmd.Backward, LineMode: m.lineMode}, count}
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward}, count}
	case 'j':
		g.Commands <- cmd.Repeat{cmd.AdjustSelection{Dir: cmd.Down, LineMode: m.lineMode}, count}
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
	case 'k':
		g.Commands <- cmd.Repeat{cmd.AdjustSelection{Dir: cmd.Up, LineMode: m.lineMode}, count}
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
	case 'l':
		g.Commands <- cmd.Repeat{cmd.AdjustSelection{Dir: cmd.Forward, LineMode: m.lineMode}, count}
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Forward}, count}
	case 'd':
		start, end := view.GetVisualSelection(v)
		v.Buffer().DeleteRange(start, end)
		m.editor.SetMode(NewNormalMode(m.editor))
	case 'v':
		m.editor.SetMode(NewNormalMode(m.editor))
	case 'V':
		if m.lineMode {
			m.editor.SetMode(NewNormalMode(m.editor))
		} else {
			v.VisualRange().SetStartOffset(0)
			v.VisualRange().SetEndOffset(len(c.Line.Data))
		}
	}

	// FIXME: there must be a better way of doing this
	// trigger a re-draw
	g.Resize()

	m.count = ""
}

func (m *visualMode) Exit() {
	v := m.editor.ActiveView()
	v.SetVisualRange(nil)
}
