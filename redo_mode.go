package main

import (
	"github.com/nsf/termbox-go"
)

type redo_mode struct {
	stub_overlay_mode
	editor *editor
}

func init_redo_mode(editor *editor) redo_mode {
	r := redo_mode{editor: editor}
	return r
}

func (r redo_mode) onKey(ev *termbox.Event) {
	g := r.editor
	v := g.active.leaf
	if ev.Mod == 0 && ev.Key == termbox.KeyCtrlSlash {
		v.on_vcommand(vcommand_redo, 0)
		return
	}

	g.set_overlay_mode(nil)
	g.onKey(ev)
}
