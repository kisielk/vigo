package main

import (
	"github.com/nsf/termbox-go"
)

type autocomplete_mode struct {
	stub_overlay_mode
	editor     *editor
	origin     cursor_location
	proposals  []ac_proposal
	prefix_len int
	current    int
}

func init_autocomplete_mode(editor *editor) *autocomplete_mode {
	view := editor.active.leaf

	a := new(autocomplete_mode)
	a.editor = editor
	a.origin = view.cursor
	a.proposals, a.prefix_len = local_ac(view)
	a.current = -1
	a.substitute_next()
	return a
}

func (a *autocomplete_mode) substitute_next() {
	view := a.editor.active.leaf
	if a.current != -1 {
		// undo previous substitution
		view.undo()
		a.editor.set_status("") // hide undo status message
	}

	a.current++
	if a.current >= len(a.proposals) {
		a.current = -1
		a.editor.set_status("No further expansions found")
		return
	}

	// create a new one
	c := view.cursor
	view.finalize_action_group()
	if a.prefix_len != 0 {
		c.move_one_word_backward()
		wlen := a.origin.boffset - c.boffset
		view.action_delete(c, wlen)
	}
	newword := clone_byte_slice(a.proposals[a.current].content)
	view.action_insert(c, newword)
	view.last_vcommand = vcommand_none
	view.dirty = dirty_everything
	c.boffset += len(newword)
	view.move_cursor_to(c)
	view.finalize_action_group()
}

func (a *autocomplete_mode) onKey(ev *termbox.Event) {
	g := a.editor
	if ev.Mod&termbox.ModAlt != 0 && ev.Ch == '/' {
		a.substitute_next()
		return
	}

	g.set_overlay_mode(nil)
	g.onKey(ev)
}
