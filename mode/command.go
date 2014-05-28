package mode

import (
	"bytes"
	"fmt"
	"github.com/kisielk/vigo/editor"
	"github.com/nsf/termbox-go"
	"strconv"
	"strings"
)

type CommandMode struct {
	editor.Overlay
	editor *editor.Editor
	mode   editor.EditorMode
	buffer *bytes.Buffer
}

func NewCommandMode(editor *editor.Editor, mode editor.EditorMode) CommandMode {
	m := CommandMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	return m
}

func (m CommandMode) NeedsCursor() bool {
	return true
}

func (m CommandMode) CursorPosition() (int, int) {
	e := m.editor
	return m.buffer.Len() + 1, e.Height() - 1
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
			m.editor.SetStatus(":" + c)
		}
		m.editor.SetMode(m.mode)
	case termbox.KeySpace:
		m.buffer.WriteRune(' ')
	default:
		m.buffer.WriteRune(ev.Ch)
	}
}

func (m CommandMode) Exit() {
}

func (m CommandMode) Draw() {
	m.editor.DrawStatus([]byte(":" + m.buffer.String()))
}

// Interpret command and apply changes to editor.
func execCommand(e *editor.Editor, command string) error {
	fields := strings.Fields(command)
	cmd, args := fields[0], fields[1:]
	switch cmd {
	case "q":
		e.Quit()
	case "w":
		b := e.ActiveView().Buffer()
		switch len(args) {
		case 0:
			b.Save()
		case 1:
			b.SaveAs(args[0])
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
		buffer, err := e.NewBufferFromFile(filename)
		if err != nil {
			return err
		}
		e.ActiveView().Attach(buffer)
	}

	if lineNum, err := strconv.Atoi(cmd); err == nil {
		// cmd is a number, we should move to that line
		e.ActiveView().MoveCursorToLine(lineNum)
	}

	return nil
}
