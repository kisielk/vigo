package main

import (
	"github.com/kisielk/vigo/editor"
	"github.com/nsf/termbox-go"
	"os"
)

func main() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	e := editor.NewEditor(os.Args[1:])
	e.Resize()
	e.Draw()
	termbox.SetCursor(e.CursorPosition())
	termbox.Flush()
	go func() {
		for {
			e.Events <- termbox.PollEvent()
		}
	}()
	if err := e.Loop(); err != editor.ErrQuit {
		panic(err)
	}
}
