package main

import (
	"github.com/nsf/termbox-go"
	"os"
)

func main() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	editor := newEditor(os.Args[1:])
	editor.resize()
	editor.draw()
	termbox.SetCursor(editor.cursorPosition())
	termbox.Flush()
	go func() {
		for {
			editor.events <- termbox.PollEvent()
		}
	}()
	if err := editor.Loop(); err != errQuit {
		panic(err)
	}
}

// this is a structure which represents a key press, used for keyboard macros
type keyEvent struct {
	mod termbox.Modifier
	_   [1]byte
	key termbox.Key
	ch  rune
}

func createKeyEvent(ev *termbox.Event) keyEvent {
	return keyEvent{
		mod: ev.Mod,
		key: ev.Key,
		ch:  ev.Ch,
	}
}

func (k keyEvent) toTermboxEvent() termbox.Event {
	return termbox.Event{
		Type: termbox.EventKey,
		Mod:  k.mod,
		Key:  k.key,
		Ch:   k.ch,
	}
}
