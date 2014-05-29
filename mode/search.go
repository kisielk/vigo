package mode

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/kisielk/vigo/editor"
	cmd "github.com/kisielk/vigo/commands"
	"github.com/nsf/termbox-go"
)

type SearchMode struct {
	editor.Overlay
	editor *editor.Editor
	mode   editor.Mode
	buffer *bytes.Buffer
}

func NewSearchMode(editor *editor.Editor, mode editor.Mode) SearchMode {
	m := SearchMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	return m
}

func (m SearchMode) NeedsCursor() bool {
	return true
}

func (m SearchMode) CursorPosition() (int, int) {
	e := m.editor
	return m.buffer.Len() + 1, e.Height() - 1
}

func (m SearchMode) OnKey(ev *termbox.Event) {
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
		if err := storeSearchTerm(m.editor, c); err != nil {
			m.editor.SetStatus(fmt.Sprintf("error: %s", err))
		} else {
			m.editor.SetStatus("/" + c)
		}
		m.editor.Commands <- cmd.Search{Dir: cmd.Forward}
		m.editor.SetMode(m.mode)
	case termbox.KeySpace:
		m.buffer.WriteRune(' ')
	default:
		m.buffer.WriteRune(ev.Ch)
	}
}

func (m SearchMode) Exit() {}

func (m SearchMode) Draw() {
	m.editor.DrawStatus([]byte("/" + m.buffer.String()))
}

// Store the search term on the editor instance.
// This allows us to use it later in other commands.
func storeSearchTerm(e *editor.Editor, command string) error {
	fields := strings.Fields(command)
	term := fields[0]

	e.LastSearchTerm = term

	return nil
}
