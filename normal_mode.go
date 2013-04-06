package main

import (
	"github.com/nsf/termbox-go"

//	"github.com/nsf/tulib"
//	"strconv"
)

type NormalMode struct {
	stub_overlay_mode
	godit *godit
}

func NewNormalMode(godit *godit) NormalMode {
	m := NormalMode{godit: godit}
	m.godit.set_status("Normal")
	return m
}

func (m NormalMode) onKey(ev *termbox.Event) {
	g := m.godit
	v := g.active.leaf

	switch ev.Ch {
	case 'a':
		v.on_vcommand(vcommand_move_cursor_forward, 0)
		g.setMode(NewInsertMode(g))
	case 'A':
		v.on_vcommand(vcommand_move_cursor_end_of_line, 0)
		g.setMode(NewInsertMode(g))
	case 'h':
		v.on_vcommand(vcommand_move_cursor_backward, 0)
	case 'i':
		g.setMode(NewInsertMode(g))
	case 'j':
		v.on_vcommand(vcommand_move_cursor_next_line, 0)
	case 'k':
		v.on_vcommand(vcommand_move_cursor_prev_line, 0)
	case 'l':
		v.on_vcommand(vcommand_move_cursor_forward, 0)
	case 'x':
		v.on_vcommand(vcommand_delete_rune, 0)
	case '0':
		v.on_vcommand(vcommand_move_cursor_beginning_of_line, 0)
	case '$':
		v.on_vcommand(vcommand_move_cursor_end_of_line, 0)
	}
}
