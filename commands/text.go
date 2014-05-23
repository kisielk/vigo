package commands

import (
	"github.com/kisielk/vigo/editor"
)

type InsertRune struct {
	Rune rune
}

func (r InsertRune) Apply(e *editor.Editor) {
	view := e.ActiveView()
	view.Buffer().InsertRune(view.Cursor(), r.Rune)
}

type DeleteRune struct{}

func (_ DeleteRune) Apply(e *editor.Editor) {
	view := e.ActiveView()
	view.Buffer().DeleteRune(view.Cursor())
}

type DeleteRuneBackward struct{}

func (_ DeleteRuneBackward) Apply(e *editor.Editor) {
	view := e.ActiveView()
	view.Buffer().DeleteRuneBackward(view.Cursor())
}
