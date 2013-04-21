package main

import (
	"github.com/nsf/termbox-go"
	"os"
	"runtime"
)

const (
	tabstop_length            = 8
	view_vertical_threshold   = 5
	view_horizontal_threshold = 10
)

func init() {
	runtime.GOMAXPROCS(2)
}

// this is a structure which represents a key press, used for keyboard macros
type key_event struct {
	mod termbox.Modifier
	_   [1]byte
	key termbox.Key
	ch  rune
}

func create_key_event(ev *termbox.Event) key_event {
	return key_event{
		mod: ev.Mod,
		key: ev.Key,
		ch:  ev.Ch,
	}
}

func (k key_event) to_termbox_event() termbox.Event {
	return termbox.Event{
		Type: termbox.EventKey,
		Mod:  k.mod,
		Key:  k.key,
		Ch:   k.ch,
	}
}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	editor := NewEditor(os.Args[1:])
	editor.resize()
	editor.draw()
	termbox.SetCursor(editor.cursorPosition())
	termbox.Flush()
	go func() {
		for {
			editor.Events <- termbox.PollEvent()
		}
	}()
	if err := editor.Loop(); err != ErrQuit {
		panic(err)
	}
}
