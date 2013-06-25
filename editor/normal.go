package editor

import (
	"github.com/nsf/termbox-go"
	"strconv"
)

type normalMode struct {
	editor *editor
	reps   string
}

func newNormalMode(editor *editor) *normalMode {
	m := normalMode{editor: editor}
	m.editor.setStatus("Normal")
	return &m
}

func (m *normalMode) onKey(ev *termbox.Event) {
	// Most of the key bindings are derived from those at http://elvis.the-little-red-haired-girl.org/elvisman/elvisvi.html#^Y

	g := m.editor
	v := g.active.leaf

	// Consequtive non-zero digits specify action multiplier;
	// accumulate and return. Accept zero only if it's
	// a non-starting character.
	if ('0' < ev.Ch && ev.Ch <= '9') || (ev.Ch == '0' && len(m.reps) > 0) {
		m.reps = m.reps + string(ev.Ch)
		m.editor.setStatus(m.reps)
		return
	}

	reps := parseReps(m.reps)

	switch ev.Ch {
	case 0x0:
		// TODO Cursor centering after Ctrl-U/D seems off.
		// TODO Ctrl-U and CTRL-D have configurable ranges of motion.
		switch ev.Key {
		case termbox.KeyCtrlA:
			//TODO: search for next occurrence of word under cursor
			return
		case termbox.KeyCtrlB:
			//TODO: should move a full page
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Reps: reps})
		case termbox.KeyCtrlD:
			// TODO: should move by count lines, default to 1/2 screen
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Reps: reps})
		case termbox.KeyCtrlE:
			// TODO: should move by count lines, default to 1
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Reps: reps})
		case termbox.KeyCtrlF:
			//TODO: should move a full page
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Reps: reps})
		case termbox.KeyCtrlG:
			//TODO: Show file status
			return
		case termbox.KeyCtrlH:
			// Same as 'h'
			// TODO: find a way to avoid duplication of 'h'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBackward, Reps: reps})
		case termbox.KeyCtrlJ, termbox.KeyCtrlN:
			// Same as 'j'
			// TODO: find a way to avoid duplication of 'j'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorNextLine, Reps: reps})
		case termbox.KeyCtrlL:
			// TODO: redraw screen
			return
		case termbox.KeyCtrlM:
			// TODO: move to front of next line
			return
		case termbox.KeyCtrlP:
			// same as 'k'
			// TODO: find a way to avoid duplication of 'k'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorPrevLine, Reps: reps})
		case termbox.KeyCtrlR:
			v.onVcommand(viewCommand{Cmd: vCommandRedo, Reps: reps})
		case termbox.KeyCtrlU:
			//TODO: should move by count lines, default to 1/2 screen
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Reps: reps})
		case termbox.KeyCtrlV:
			//TODO: Start visual selection
			return
		case termbox.KeyCtrlW:
			//TODO: Buffer the ctrl-W and then implement some windowing commands
			return
		case termbox.KeyCtrlX:
			//TODO: Move to column count
			return
		case termbox.KeyCtrlY:
			//TODO: should move by count lines, default to 1
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Reps: reps})
		case termbox.KeyEsc:
			//TODO: Cancel the current command
			return
		case termbox.KeySpace:
			// Same as 'l'
			// TODO: find a way to avoid duplication of 'l'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward, Reps: reps})
		}
	case 'h':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBackward, Reps: reps})
	case 'j':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorNextLine, Reps: reps})
	case 'k':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorPrevLine, Reps: reps})
	case 'l':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward, Reps: reps})
	case 'w':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordForward, Reps: reps})
	case 'e':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordEnd, Reps: reps})
	case 'b':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordBackward, Reps: reps})
	case 'x':
		v.onVcommand(viewCommand{Cmd: vCommandDeleteRune, Reps: reps})
	case 'u':
		v.onVcommand(viewCommand{Cmd: vCommandUndo, Reps: reps})
	case 'G':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfFile})
	}

	// Insert mode; record first, then repeat.
	switch ev.Ch {
	case 'a':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward})
		g.setMode(newInsertMode(g, reps))
	case 'A':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfLine})
		g.setMode(newInsertMode(g, reps))
	case 'i':
		g.setMode(newInsertMode(g, reps))
	}

	// No point repeating these commands
	switch ev.Ch {
	case '0':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBeginningOfLine})
	case '$':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfLine})
	}

	// TODO use reps to set range for command mode
	switch ev.Ch {
	case ':':
		g.setMode(newCommandMode(g, m))
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

func (m *normalMode) exit() {
}
