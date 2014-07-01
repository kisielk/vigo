package mode

import (
	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/utils"
	"github.com/kisielk/vigo/view"
	"github.com/nsf/termbox-go"
)

type visualMode struct {
	editor   *editor.Editor
	count    string
	lineMode bool
}

func NewVisualMode(e *editor.Editor, lineMode bool) visualMode {
	m := visualMode{editor: e, lineMode: lineMode}
	v := m.editor.ActiveView()
	c := v.Cursor()

	var t view.SelectionType
	if lineMode {
		m.editor.SetStatus("Visual Line")
		t = view.SelectionLine
	} else {
		m.editor.SetStatus("Visual")
		t = view.SelectionChar
	}

	sel := view.Selection{Type: t}
	sel.Range.Start = c
	sel.Range.End = c

	v.SetSelection(sel)

	return m
}

func (m *visualMode) Enter(e *editor.Editor) {
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

	switch ev.Key {
	case termbox.KeyEsc:
		m.editor.SetMode(NewNormalMode(m.editor))
	}

	switch ev.Ch {
	case 'h':
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward}, count}
	case 'j':
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
	case 'k':
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
	case 'l':
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Forward}, count}
	case 'd':
		r := v.Selection().EffectiveRange()
		v.Buffer().DeleteRange(r.Start, r.End)
		m.editor.SetMode(NewNormalMode(m.editor))
	case 'v':
		m.editor.SetMode(NewNormalMode(m.editor))
	case 'V':
		if m.lineMode {
			m.editor.SetMode(NewNormalMode(m.editor))
		} else {
			sel := v.Selection()
			sel.Type = view.SelectionLine
			v.SetSelection(sel)
		}
	}

	m.Reset()
}

func (m *visualMode) Reset() {
  m.count = ""
}

func (m *visualMode) Exit() {
	v := m.editor.ActiveView()
	v.SetSelection(view.Selection{Type: view.SelectionNone})
}
