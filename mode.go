package main

import (
	"github.com/nsf/termbox-go"
	"strconv"
)

type EditorMode interface {
	OnKey(ev *termbox.Event)
	Exit()
}
type VisualMode struct {
	editor *editor
	reps   string
}

func NewVisualMode(editor *editor) *VisualMode {
	m := VisualMode{editor: editor}
	m.editor.SetStatus("Visual")
	return &m
}

func (m *VisualMode) OnKey(ev *termbox.Event) {
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

	switch ev.Ch {
	case 0x0:
		switch ev.Key {
		case termbox.KeySpace:
			v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_forward, Reps: reps})
		case termbox.KeyCtrlR:
			v.on_vcommand(ViewCommand{Cmd: vcommand_redo, Reps: reps})
		}
	case 's':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_backward, Reps: reps})
	case 'n':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_next_line, Reps: reps})
	case 'e':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_prev_line, Reps: reps})
	case 't':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_forward, Reps: reps})
	case 'w':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_word_forward, Reps: reps})
	case 'b':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_word_backward, Reps: reps})
	case 'x':
		v.on_vcommand(ViewCommand{Cmd: vcommand_delete_rune, Reps: reps})
	case 'u':
		v.on_vcommand(ViewCommand{Cmd: vcommand_undo, Reps: reps})
	}

	// Insert mode; record first, then repeat.
	switch ev.Ch {
	case 'a':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_forward})
		g.SetMode(NewInsertMode(g, reps))
	case 'A':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_end_of_line})
		g.SetMode(NewInsertMode(g, reps))
	case 'i':
		g.SetMode(NewInsertMode(g, reps))
	}

	// No point repeating these commands
	switch ev.Ch {
	case '0':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_beginning_of_line})
	case '$':
		v.on_vcommand(ViewCommand{Cmd: vcommand_move_cursor_end_of_line})
	}

	if ev.Ch == 0x0 {
		switch ev.Key {
		// TODO Cursor centering after Ctrl-U/D seems off.
		// TODO Ctrl-U and CTRL-D have configurable ranges of motion.
		case termbox.KeyCtrlU, termbox.KeyCtrlB:
			v.on_vcommand(ViewCommand{Cmd: vcommand_move_view_half_backward, Reps: reps})
		case termbox.KeyCtrlD, termbox.KeyCtrlF:
			v.on_vcommand(ViewCommand{Cmd: vcommand_move_view_half_forward, Reps: reps})
		}
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

func (m *VisualMode) Exit() {
}
