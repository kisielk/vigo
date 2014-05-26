package main

import (
	"os"

	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/mode"
	"github.com/nsf/termbox-go"
)

func main() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	e := editor.NewEditor(os.Args[1:])
	e.SetMode(mode.NewNormalMode(e))
	e.Resize()
	e.Draw()
	termbox.SetCursor(e.CursorPosition())
	termbox.Flush()
	go func() {
		for {
			e.UIEvents <- termbox.PollEvent()
		}
	}()
	if err := e.Loop(); err != editor.ErrQuit {
		panic(err)
	}
}
