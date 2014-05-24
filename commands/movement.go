package commands

import (
	"github.com/kisielk/vigo/editor"
)

type Dir int

const (
	Forward  Dir = 0
	Backward Dir = 1
)

type MoveWord struct {
	Dir Dir
}

func (m MoveWord) Apply(e *editor.Editor) {
	// moveCursorWordForward
	v := e.ActiveView()
	c := v.Cursor()

	switch m.Dir {
	case Forward:
		ok := c.NextWord()
		if !ok {
			e.SetStatus("End of file")
			return
		}
	case Backward:
		ok := c.PrevWord()
		if !ok {
			e.SetStatus("Beginning of file")
			return
		}
	}

	v.MoveCursorTo(c)
}

type MoveRune struct {
	Dir  Dir
	Wrap bool
}

func (m MoveRune) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()

	switch m.Dir {
	case Forward:
		if c.LastLine() && c.EOL() {
			v.SetStatus("End of file")
			return
		}
		c.NextRune(m.Wrap)
	case Backward:
		if c.FirstLine() && c.BOL() {
			v.SetStatus("Beginning of file")
			return
		}
		c.PrevRune(m.Wrap)
	}

	v.MoveCursorTo(c)
}
