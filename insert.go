package main

import (
	"github.com/nsf/termbox-go"
)

type InsertMode struct {
	editor *editor
	reps   int
}

func NewInsertMode(editor *editor, reps int) InsertMode {
	m := InsertMode{editor: editor}
	m.editor.SetStatus("Insert")
	m.reps = reps
	return m
}

func (m InsertMode) OnKey(ev *termbox.Event) {
	g := m.editor
	v := g.active.leaf

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.SetMode(NewVisualMode(g))
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		v.on_vcommand(ViewCommand{Cmd: vcommand_delete_rune_backward})
	case termbox.KeyDelete, termbox.KeyCtrlD:
		v.on_vcommand(ViewCommand{Cmd: vcommand_delete_rune})
	case termbox.KeySpace:
		v.on_vcommand(ViewCommand{Cmd: vcommand_insert_rune, Rune: ' '})
	case termbox.KeyEnter, termbox.KeyCtrlJ:
		c := '\n'
		if ev.Key == termbox.KeyEnter {
			// we use '\r' for <enter>, because it doesn't cause
			// autoindent
			c = '\r'
		}
		v.on_vcommand(ViewCommand{Cmd: vcommand_insert_rune, Rune: c})
	default:
		if ev.Ch != 0 {
			v.on_vcommand(ViewCommand{Cmd: vcommand_insert_rune, Rune: ev.Ch})
		}
	}
}

func (m InsertMode) Exit() {
	// repeat action specified number of times
	for i := 0; i < m.reps-1; i++ {
		g := m.editor
		v := g.active.leaf
		a := v.buf.history.last_action()
		a.do(v, a.what)
	}
}
