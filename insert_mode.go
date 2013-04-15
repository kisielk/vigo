package main

import (
	"github.com/nsf/termbox-go"
)

type InsertMode struct {
	stub_overlay_mode
	editor *editor
	reps   int
}

func NewInsertMode(editor *editor, reps int) InsertMode {
	m := InsertMode{editor: editor}
	m.editor.set_status("Insert")
	m.reps = reps
	return m
}

func (m InsertMode) OnKey(ev *termbox.Event) {
	g := m.editor
	v := g.active.leaf

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.setMode(NewNormalMode(g))
	default:
		v.onKey(ev)
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
