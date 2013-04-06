package main

import (
	"github.com/nsf/termbox-go"
)

type InsertMode struct {
	stub_overlay_mode
	godit *godit
}

func NewInsertMode(godit *godit) InsertMode {
	m := InsertMode{godit: godit}
	m.godit.set_status("Insert")
	return m
}

func (m InsertMode) onKey(ev *termbox.Event) {
	g := m.godit
	v := g.active.leaf

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.setMode(NewNormalMode(g))
	default:
		v.onKey(ev)
	}
}
