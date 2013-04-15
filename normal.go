package main

import (
	"github.com/nsf/termbox-go"
	"strconv"
)

type NormalMode struct {
	stub_overlay_mode
	editor *editor
	reps   string
}

func NewNormalMode(editor *editor) *NormalMode {
	m := NormalMode{editor: editor}
	m.editor.SetStatus("Normal")
	return &m
}

func (m *NormalMode) OnKey(ev *termbox.Event) {
	g := m.editor
	v := g.active.leaf

	// Consequtive non-zero digits specify action multiplier;
	// accumulate and return. Accept zero only if it's
	// a non-starting character.
	if ('0' < ev.Ch && ev.Ch <= '9') || (ev.Ch == '0' && len(m.reps) > 0) {
		m.reps = m.reps + string(ev.Ch)
		m.editor.SetStatus(m.reps)
		return
	}

	reps := parseReps(m.reps)

	for i := 0; i < reps; i++ {
		switch ev.Ch {
		case 'h':
			v.on_vcommand(vcommand_move_cursor_backward, 0)
		case 'j':
			v.on_vcommand(vcommand_move_cursor_next_line, 0)
		case 'k':
			v.on_vcommand(vcommand_move_cursor_prev_line, 0)
		case 'l':
			v.on_vcommand(vcommand_move_cursor_forward, 0)
		case 'w':
			v.on_vcommand(vcommand_move_cursor_word_forward, 0)
		case 'b':
			v.on_vcommand(vcommand_move_cursor_word_backward, 0)
		case 'x':
			v.on_vcommand(vcommand_delete_rune, 0)
		}
	}

	// Action producers; recent action needs to replayed reps times.
	switch ev.Ch {
	case 'a':
		v.on_vcommand(vcommand_move_cursor_forward, 0)
		g.SetMode(NewInsertMode(g, reps))
	case 'A':
		v.on_vcommand(vcommand_move_cursor_end_of_line, 0)
		g.SetMode(NewInsertMode(g, reps))
	case 'i':
		g.SetMode(NewInsertMode(g, reps))
	}

	// No point repeating these commands
	switch ev.Ch {
	case '0':
		v.on_vcommand(vcommand_move_cursor_beginning_of_line, 0)
	case '$':
		v.on_vcommand(vcommand_move_cursor_end_of_line, 0)
	}

	// TODO use reps to set range for command mode
	switch ev.Ch {
	case ':':
		g.SetMode(NewCommandMode(g, m))
	}

	// Reset repetitions
	m.reps = ""
}

// Parse action multiplier from a string.
func parseReps(s string) int {
	var n int64 = 1
	var err error
	if len(s) > 0 {
		n, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			panic("could not parse action multiplier")
		}
	}
	return int(n)
}

func (m *NormalMode) Exit() {
}
