package main

import (
	"github.com/nsf/termbox-go"
)

type CommandMode struct {
	editor *editor
}

func NewCommandMode(editor *editor) CommandMode {
	m := CommandMode{editor: editor}
	m.editor.set_status("Command")
	return m
}

func (m CommandMode) OnKey(ev *termbox.Event) {

}

func (m CommandMode) Exit() {

}
