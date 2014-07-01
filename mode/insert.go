package mode

import (
	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/nsf/termbox-go"
	"time"
)

type InsertMode struct {
	editor *editor.Editor
	count  int
}

func (m *InsertMode) Reset() {
}

func (m *InsertMode) Enter(e *editor.Editor) {
}

func (m *InsertMode) Exit() {
	// repeat action specified number of times
	v := m.editor.ActiveView()
	b := v.Buffer()
	for i := 0; i < m.count-1; i++ {
		a := b.History.LastAction()
		a.Apply(b)
	}
}

func NewInsertMode(editor *editor.Editor, count int) InsertMode {
	m := InsertMode{editor: editor}
	m.editor.SetStatus("Insert")
	m.count = count
	return m
}

func (m *InsertMode) OnKey(ev *termbox.Event) {
	var newMode editor.Mode
	if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlC {
		newMode = NewNormalMode(m.editor)
		m.editor.SetMode(&newMode)
		return
	}

	var eventChan = make(chan *termbox.Event)

	g := m.editor

	go m.getCommand(g.Commands, eventChan)

	eventChan <- ev

	close(eventChan)
}

func (m *InsertMode) getCommand(commandChan chan<- editor.Command, eventChan <-chan *termbox.Event) {

	var ev *termbox.Event

	select {
	case ev = <-eventChan:
		switch ev.Key {
		case termbox.KeyBackspace, termbox.KeyBackspace2:
			commandChan <- cmd.DeleteRuneBackward{}
		case termbox.KeyDelete, termbox.KeyCtrlD:
			commandChan <- cmd.DeleteRune{}
		case termbox.KeySpace:
			commandChan <- cmd.InsertRune{' '}
		case termbox.KeyEnter:
			// we use '\r' for <enter>, because it doesn't cause autoindent
			commandChan <- cmd.InsertRune{'\r'}
		case termbox.KeyCtrlJ:
			commandChan <- cmd.InsertRune{'\n'}
		default:
			if ev.Ch != 0 {
				commandChan <- cmd.InsertRune{ev.Ch}
			}
		}
	case <-time.After(time.Nanosecond * 500 * 1000000):
		commandChan <- cmd.ResetMode{Mode: m}
	}
	close(commandChan)
}

