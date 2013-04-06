package main

import (
	"github.com/nsf/termbox-go"
)

type CommandMode struct {
	stub_overlay_mode
	editor *editor
}

func NewCommandMode(editor *editor) CommandMode {
	m := CommandMode{editor: editor}
	m.editor.set_status("Command")
	return m
}

func (m CommandMode) onKey(ev *termbox.Event) {

}
