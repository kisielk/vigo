package main

import (
	"bytes"
	"fmt"
	"github.com/nsf/termbox-go"
	"strings"
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
		m.editor.SetMode(m.mode)
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		l := m.buffer.Len()
		if l > 0 {
			m.buffer.Truncate(l - 1)
		}
	case termbox.KeyEnter:
		c := m.buffer.String()
		if err := execCommand(m.editor, c); err != nil {
			m.editor.SetStatus(fmt.Sprintf("error: %s", err))
		} else {
			m.editor.SetStatus(c)
		}
		m.editor.SetMode(m.mode)
	case termbox.KeySpace:
		m.buffer.WriteRune(' ')
	default:
		m.buffer.WriteRune(ev.Ch)
	}
	m.render()
}

func (m CommandMode) Exit() {
}

func (m CommandMode) render() {
	m.editor.SetStatus(":" + m.buffer.String() + cursorChar)
}

// Interpret command and apply changes to editor.
func execCommand(e *editor, command string) error {
	fields := strings.Fields(command)
	cmd, args := fields[0], fields[1:]
	switch cmd {
	case "q":
		e.Quit()
	case "w":
		if len(args) == 0 {
			e.active.leaf.buf.save()
		} else if len(args) == 1 {
			e.active.leaf.buf.save_as(args[0])
		} else {
			return fmt.Errorf("too many arguments to :w")
		}
	case "e":
		if len(args) != 1 {
			return fmt.Errorf("wrong number of arguments for :e")
		}
	}
	return nil
}
