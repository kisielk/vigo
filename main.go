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
