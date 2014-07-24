package commands

import (
	"github.com/kisielk/vigo/editor"
)
type ResetMode struct {
	Mode editor.Mode
}

func (m ResetMode) Apply(e *editor.Editor) {
	m.Mode.Reset()
}

/*
type EnterNormalMode struct{}

func (_ EnterNormalMode) Apply(e *editor.Editor) {
	e.SetMode(editor.NewNormalMode(e))
}

type EnterInsertMode struct {
	Count int // Number of times to repeat the inserted text
}

func (m EnterInsertMode) Apply(e *editor.Editor) {
	e.SetMode(editor.NewInsertMode(e, m.Count))
}
*/
