package commands

import (
	"github.com/kisielk/vigo/editor"
)

type Undo struct {
	Count int
}

func (u Undo) Apply(e *editor.Editor) {
	for i := 0; i < u.Count; i++ {
		e.ActiveView().Buffer().Undo()
	}
}

type Redo struct {
	Count int
}

func (r Redo) Apply(e *editor.Editor) {
	for i := 0; i < r.Count; i++ {
		e.ActiveView().Buffer().Redo()
	}
}
