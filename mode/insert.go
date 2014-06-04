package mode

import (
	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/nsf/termbox-go"
)

type insertMode struct {
	editor *editor.Editor
	count  int
}

func NewInsertMode(editor *editor.Editor, count int) insertMode {
	m := insertMode{editor: editor}
	m.editor.SetStatus("Insert")
	m.count = count
	return m
}

func (m insertMode) Enter(editor *editor.Editor) {
}

func (m insertMode) OnKey(ev *termbox.Event) {
	g := m.editor

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.SetMode(NewNormalMode(g))
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		g.Commands <- cmd.DeleteRuneBackward{}
	case termbox.KeyDelete, termbox.KeyCtrlD:
		g.Commands <- cmd.DeleteRune{}
	case termbox.KeySpace:
		g.Commands <- cmd.InsertRune{' '}
	case termbox.KeyEnter:
		// we use '\r' for <enter>, because it doesn't cause autoindent
		g.Commands <- cmd.InsertRune{'\r'}
	case termbox.KeyCtrlJ:
		g.Commands <- cmd.InsertRune{'\n'}
	default:
		if ev.Ch != 0 {
			g.Commands <- cmd.InsertRune{ev.Ch}
		}
	}
}

func (m insertMode) Exit() {
	// repeat action specified number of times
	v := m.editor.ActiveView()
	b := v.Buffer()
	for i := 0; i < m.count-1; i++ {
		a := b.History.LastAction()
		a.Apply(b)
	}
}
