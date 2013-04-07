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
	m.editor.SetStatus("Insert")
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
}
