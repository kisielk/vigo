package editor

import (
	"github.com/nsf/termbox-go"
	"syscall"
)

func suspend(e *Editor) {
	// finalize termbox
	termbox.Close()

	// suspend the process
	pid := syscall.Getpid()
	tid := syscall.Gettid()
	err := syscall.Tgkill(pid, tid, syscall.SIGSTOP)
	if err != nil {
		panic(err)
	}

	// reset the state so we can get back to work again
	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputAlt)
	e.Resize()
}
