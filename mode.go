package main

import (
	"github.com/nsf/termbox-go"
)

type EditorMode interface {
	OnKey(ev *termbox.Event)
	Exit()
}
