package main

import (
	"github.com/nsf/termbox-go"
)

type InsertMode struct {
	stub_overlay_mode
	editor *editor
}

func NewInsertMode(editor *editor) InsertMode {
	m := InsertMode{editor: editor}
	m.editor.set_status("Insert")
	return m
}

func (m InsertMode) onKey(ev *termbox.Event) {
	g := m.editor
	v := g.active.leaf

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.setMode(NewNormalMode(g))
	default:
		v.onKey(ev)
	}
}
