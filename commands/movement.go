package commands

import (
	"github.com/kisielk/vigo/buffer"
	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/utils"
)

type Dir int

const (
	Forward  Dir = 0
	Backward Dir = 1
	Up       Dir = 2
	Down     Dir = 3
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
			v.SetStatus("End of line")
			return
		}
	case Backward:
		if !c.PrevRune(m.Wrap) {
			v.SetStatus("Beginning of line")
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
			v.SetStatus("End of file")
			return
		}
	case Backward:
		if !c.PrevWord() {
			v.SetStatus("Beginning of file")
			return
		}
	}

	v.MoveCursorTo(c)
}

type MoveWordEnd struct{}

func (m MoveWordEnd) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()
	ok := c.EndWord()
	v.MoveCursorTo(c)
	// FIXME Message is never printed
	if !ok {
		v.SetStatus("End of buffer")
	}
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

type MoveBOL struct{}

func (m MoveBOL) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()
	c.MoveBOL()
	v.MoveCursorTo(c)
}

// Front-of-line is defined as the first non-space character in the line.
type MoveFOL struct{}

func (m MoveFOL) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()
	pos := utils.IndexFirstNonSpace(c.Line.Data)
	c.Boffset = pos
	v.MoveCursorTo(c)
}

type MoveEOL struct{}

func (m MoveEOL) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()
	c.MoveEOL()
	v.MoveCursorTo(c)
}

type MoveEOF struct{}

func (m MoveEOF) Apply(e *editor.Editor) {
	v := e.ActiveView()
	b := v.Buffer()
	c := buffer.Cursor{
		Line:    b.LastLine,
		LineNum: b.NumLines,
		Boffset: b.LastLine.Len(),
	}
	v.MoveCursorTo(c)
}

type MoveView struct {
	// TODO use Repeat{} rather than lines argument?
	Lines int
	Dir   Dir
}

func (m MoveView) Apply(e *editor.Editor) {
	v := e.ActiveView()
	switch m.Dir {
	case Forward:
		v.MoveViewLines(m.Lines)
	case Backward:
		v.MoveViewLines(-m.Lines)
	}
}

type NearestHSplit struct {
	Dir Dir
}

func (m NearestHSplit) Apply(e *editor.Editor) {
	n := e.ActiveViewNode()
	var d int
	switch m.Dir {
	case Forward:
		d = 1
	case Backward:
		d = -1
	}
	if k := n.NearestHSplit(d); k != nil {
		e.SetActiveViewNode(k)
	}
}

type NearestVSplit struct {
	Dir Dir
}

func (m NearestVSplit) Apply(e *editor.Editor) {
	n := e.ActiveViewNode()
	var d int
	switch m.Dir {
	case Forward:
		d = 1
	case Backward:
		d = -1
	}
	if k := n.NearestVSplit(d); k != nil {
		e.SetActiveViewNode(k)
	}
}
