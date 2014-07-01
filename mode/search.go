package mode

import (
	"bytes"
	"strings"

	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/nsf/termbox-go"
)

type SearchMode struct {
	editor.Overlay
	editor *editor.Editor
	mode   editor.Mode
	buffer *bytes.Buffer
}

func NewSearchMode(editor *editor.Editor, mode editor.Mode) *SearchMode {
	m := SearchMode{editor: editor, mode: mode, buffer: &bytes.Buffer{}}
	return &m
}

func (m *SearchMode) Enter(e *editor.Editor) {
}

func (m *SearchMode) NeedsCursor() bool {
	return true
}

func (m *SearchMode) CursorPosition() (int, int) {
	e := m.editor
	return m.buffer.Len() + 1, e.Height() - 1
}

func (m *SearchMode) Reset () {
}

func (m *SearchMode) OnKey(ev *termbox.Event) {
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
		term := strings.Fields(c)[0]
		storeSearchTerm(m.editor, term)
		m.editor.Commands <- cmd.Search{Dir: cmd.Forward}
		m.editor.SetMode(m.mode)
	case termbox.KeySpace:
		m.buffer.WriteRune(' ')
	default:
		m.buffer.WriteRune(ev.Ch)
	}
}

func (m *SearchMode) Exit() {}

func (m *SearchMode) Draw() {
	m.editor.DrawStatus([]byte("/" + m.buffer.String()))
}

// Store the search term on the editor instance.
// This allows us to use it later in other commands.
func storeSearchTerm(e *editor.Editor, term string) {
	// don't do anything if no term is given
	if term == "" {
		return
	}
	e.LastSearchTerm = term
	e.ActiveView().SetHighlightBytes([]byte(term))
}
