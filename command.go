package main

import (
	"bytes"
	"fmt"
	"github.com/nsf/termbox-go"
	"strings"
)

type commandMode struct {
	Overlay
	editor *editor
	mode   editorMode
	buffer *bytes.Buffer
}

func newCommandMode(editor *editor, mode editorMode) commandMode {
	m := commandMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	return m
}

func (m commandMode) needsCursor() bool {
	return true
}

func (m commandMode) cursorPosition() (int, int) {
	e := m.editor
	r := e.uiBuf.Rect
	return m.buffer.Len() + 1, r.Height - 1
}

func (m commandMode) onKey(ev *termbox.Event) {
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
		if err := execCommand(m.editor, c); err != nil {
			m.editor.setStatus(fmt.Sprintf("error: %s", err))
		} else {
			m.editor.setStatus(":" + c)
		}
		m.editor.setMode(m.mode)
	case termbox.KeySpace:
		m.buffer.WriteRune(' ')
	default:
		m.buffer.WriteRune(ev.Ch)
	}
}

func (m commandMode) exit() {
}

func (m commandMode) draw() {
	m.editor.drawStatus([]byte(":" + m.buffer.String()))
}

// Interpret command and apply changes to editor.
func execCommand(e *editor, command string) error {
	fields := strings.Fields(command)
	cmd, args := fields[0], fields[1:]
	switch cmd {
	case "q":
		e.quit()
	case "w":
		switch len(args) {
		case 0:
			e.active.leaf.buf.save()
		case 1:
			e.active.leaf.buf.saveAs(args[0])
		default:
			return fmt.Errorf("too many arguments to :w")
		}
	case "e":
		var filename string
		switch len(args) {
		case 0:
			return fmt.Errorf("TODO re-read current file, if any")
		case 1:
			filename = args[0]
		default:
			return fmt.Errorf("too many arguments for :e")
		}

		// TODO: Don't replace the current buffer if it has been modified
		buffer, err := e.newBufferFromFile(filename)
		if err != nil {
			return err
		}
		e.active.leaf.attach(buffer)
	}
	return nil
}
