package commands

import (
	"github.com/kisielk/vigo/editor"
)

type Undo struct{}

func (u Undo) Apply(e *editor.Editor) {
	e.ActiveView().Buffer().Undo()
}

type Redo struct{}

func (r Redo) Apply(e *editor.Editor) {
	e.ActiveView().Buffer().Redo()
}
