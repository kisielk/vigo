package commands

import (
	"github.com/kisielk/vigo/editor"
)

type MoveWord struct {
}

func (m MoveWord) Apply(e *editor.Editor) {
	// moveCursorWordForward
	v := e.ActiveView()
	c := v.Cursor()
	ok := c.NextWord()
	if !ok {
		e.SetStatus("End of buffer")
		return
	}
	v.MoveCursorTo(c)
}
