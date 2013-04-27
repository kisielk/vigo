package editor

import (
	"github.com/nsf/termbox-go"
)

type insertMode struct {
	editor *editor
	reps   int
}

func newInsertMode(editor *editor, reps int) insertMode {
	m := insertMode{editor: editor}
	m.editor.setStatus("Insert")
	m.reps = reps
	return m
}

func (m insertMode) onKey(ev *termbox.Event) {
	g := m.editor
	v := g.active.leaf

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.setMode(newNormalMode(g))
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		v.onVcommand(viewCommand{Cmd: vCommandDeleteRuneBackward})
	case termbox.KeyDelete, termbox.KeyCtrlD:
		v.onVcommand(viewCommand{Cmd: vCommandDeleteRune})
	case termbox.KeySpace:
		v.onVcommand(viewCommand{Cmd: vCommandInsertRune, Rune: ' '})
	case termbox.KeyEnter, termbox.KeyCtrlJ:
		c := '\n'
		if ev.Key == termbox.KeyEnter {
			// we use '\r' for <enter>, because it doesn't cause
			// autoindent
			c = '\r'
		}
		v.onVcommand(viewCommand{Cmd: vCommandInsertRune, Rune: c})
	default:
		if ev.Ch != 0 {
			v.onVcommand(viewCommand{Cmd: vCommandInsertRune, Rune: ev.Ch})
		}
	}
}

func (m insertMode) exit() {
	// repeat action specified number of times
	for i := 0; i < m.reps-1; i++ {
		g := m.editor
		v := g.active.leaf
		a := v.buf.History.LastAction()
		a.Apply(v.buf)
	}
}
