package main

import (
	"github.com/nsf/termbox-go"
)

type CommandMode struct {
	stub_overlay_mode
	godit *godit
}

func NewCommandMode(godit *godit) CommandMode {
	m := CommandMode{godit: godit}
	m.godit.set_status("Command")
	return m
}

func (m CommandMode) on_key(ev *termbox.Event) {

}
