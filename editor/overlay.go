package editor

import (
	"github.com/nsf/termbox-go"
)

type Overlay interface {
	needsCursor() bool
	cursorPosition() (int, int)
	onResize(ev *termbox.Event)
	draw()
}
