package main

import (
	"github.com/nsf/termbox-go"
)

type macro_repeat_mode struct {
	stub_overlay_mode
	editor *editor
}

func init_macro_repeat_mode(editor *editor) macro_repeat_mode {
	m := macro_repeat_mode{editor: editor}
	editor.set_overlay_mode(nil)
	m.editor.replay_macro()
	m.editor.set_status("(Type e to repeat macro)")
	return m
}

func (m macro_repeat_mode) onKey(ev *termbox.Event) {
	g := m.editor
	if ev.Mod == 0 && ev.Ch == 'e' {
		g.set_overlay_mode(nil)
		g.replay_macro()
		g.set_overlay_mode(m)
		g.set_status("(Type e to repeat macro)")
		return
	}

	g.set_overlay_mode(nil)
	g.onKey(ev)
}
