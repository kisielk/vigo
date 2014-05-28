package editor

import (
	"github.com/nsf/termbox-go"
)

type Overlay interface {
	NeedsCursor() bool
	CursorPosition() (int, int)
	OnResize(ev *termbox.Event)
	Draw()
}
