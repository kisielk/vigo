package commands

import (
	"github.com/kisielk/vigo/editor"
)

type Dir int

const (
	Forward  Dir = 0
	Backward Dir = 1
)

type MoveRune struct {
	Dir  Dir
	Wrap bool
}

func (m MoveRune) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()

	switch m.Dir {
	case Forward:
		if !c.NextRune(m.Wrap) {
			v.SetStatus("End of file")
			return
		}
	case Backward:
		if !c.PrevRune(m.Wrap) {
			v.SetStatus("Beginning of file")
			return
		}
	}

	v.MoveCursorTo(c)
}

type MoveWord struct {
	Dir Dir
}

func (m MoveWord) Apply(e *editor.Editor) {
	// moveCursorWordForward
	v := e.ActiveView()
	c := v.Cursor()

	switch m.Dir {
	case Forward:
		if !c.NextWord() {
			e.SetStatus("End of file")
			return
		}
	case Backward:
		if !c.PrevWord() {
			e.SetStatus("Beginning of file")
			return
		}
	}

	v.MoveCursorTo(c)
}

type MoveLine struct {
	Dir Dir
}

func (m MoveLine) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()

	switch m.Dir {
	case Forward:
		if !c.NextLine() {
			v.SetStatus("End of file")
			return
		}
	case Backward:
		if !c.PrevLine() {
			v.SetStatus("Beginning of file")
			return
		}
	}

	v.MoveCursorTo(c)
}
