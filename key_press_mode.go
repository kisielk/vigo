package main

import (
	"github.com/nsf/termbox-go"
)

type key_press_mode struct {
	stub_overlay_mode
	editor  *editor
	actions map[rune]func()
	def     rune
	prompt  string
}

func init_key_press_mode(editor *editor, actions map[rune]func(), def rune, prompt string) *key_press_mode {
	k := new(key_press_mode)
	k.editor = editor
	k.actions = actions
	k.def = def
	k.prompt = prompt
	k.editor.set_status(prompt)
	return k
}

func (k *key_press_mode) onKey(ev *termbox.Event) {
	if ev.Mod != 0 {
		return
	}

	ch := ev.Ch
	if ev.Key == termbox.KeyEnter || ev.Key == termbox.KeyCtrlJ {
		ch = k.def
	}

	action, ok := k.actions[ch]
	if ok {
		action()
		k.editor.set_overlay_mode(nil)
	} else {
		k.editor.set_status(k.prompt)
	}
}
