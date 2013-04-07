package main

import (
	"bytes"
	"github.com/nsf/termbox-go"
)

const (
	cursorChar = "\u258D"
)

type CommandMode struct {
	editor *editor
	mode   EditorMode
	buffer *bytes.Buffer
}

func NewCommandMode(editor *editor, mode EditorMode) CommandMode {
	m := CommandMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	m.render()
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
		c := m.buffer.String()
		m.editor.set_status(c)
		execCommand(m.editor, c)
		m.editor.setMode(m.mode)
	default:
		m.buffer.WriteRune(ev.Ch)
	}
	m.render()
}

func (m CommandMode) Exit() {
}

func (m CommandMode) render() {
	m.editor.set_status(":" + m.buffer.String() + cursorChar)
}

// Interpret command and apply changes to editor.
func execCommand(e *editor, command string) {
	if command == "q" {
		e.Quit()
	}
}
