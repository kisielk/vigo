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

func (m InsertMode) on_key(ev *termbox.Event) {
	g := m.godit
	v := g.active.leaf

	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		g.setMode(NewNormalMode(g))
	default:
		v.on_key(ev)
	}
}
