package editor

import (
	"github.com/nsf/termbox-go"
)

type editorMode interface {
	onKey(ev *termbox.Event)
	exit()
}
