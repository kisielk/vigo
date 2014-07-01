package mode

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/kisielk/vigo/editor"
	"github.com/nsf/termbox-go"
)

type CommandMode struct {
	editor.Overlay
	editor *editor.Editor
	mode   editor.Mode
	buffer *bytes.Buffer
}

func NewCommandMode(editor *editor.Editor, mode editor.Mode) *CommandMode {
	m := CommandMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	return &m
}

func (m *CommandMode) Reset() {
	m.editor.SetMode(m.mode)
}

func (m *CommandMode) Enter(e *editor.Editor) {
}

func (m CommandMode) NeedsCursor() bool {
	return true
}

func (m CommandMode) CursorPosition() (int, int) {
	e := m.editor
	return m.buffer.Len() + 1, e.Height() - 1
}

func (m *CommandMode) OnKey(ev *termbox.Event) {
	switch ev.Key {
	case termbox.KeyEsc, termbox.KeyCtrlC:
		m.Reset()
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

	// prevent a crash if no commands are given
	if len(fields) == 0 {
		return nil
	}

	cmd, args := fields[0], fields[1:]

	switch cmd {
	case "q":
		// TODO if more than one split, close active one only.
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
	case "sp", "split":
		e.SplitHorizontally()
		// TODO file argument | shell command argument
	case "vsp", "vsplit":
		e.SplitVertically()
	case "nohls":
		e.ActiveView().ShowHighlights(false)
	case "hls":
		e.ActiveView().ShowHighlights(true)
	}

	if lineNum, err := strconv.Atoi(cmd); err == nil {
		// cmd is a number, we should move to that line
		e.ActiveView().MoveCursorToLine(lineNum)
	}

	return nil
}
