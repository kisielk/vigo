package main

import (
	"bytes"
	"github.com/nsf/termbox-go"
)

type CommandMode struct {
	editor *editor
	mode   EditorMode
	buffer *bytes.Buffer
}

func NewCommandMode(editor *editor, mode EditorMode) CommandMode {
	m := CommandMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	m.editor.set_status("Command")
	return m
}

func (m CommandMode) OnKey(ev *termbox.Event) {
	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		m.editor.setMode(m.mode)
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		l := m.buffer.Len()
		if l > 0 {
			m.buffer.Truncate(l - 1)
		}
	case termbox.KeyEnter:
		m.editor.set_status(m.buffer.String())
		m.editor.setMode(m.mode)
	default:
		m.buffer.WriteRune(ev.Ch)
	}

	m.editor.set_status(":" + m.buffer.String())
}

func (m CommandMode) Exit() {
}
