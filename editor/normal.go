package editor

import (
	"strconv"

	"github.com/nsf/termbox-go"
)

type MoveWord struct {
}

func (m MoveWord) Apply(e *Editor) {
	// moveCursorWordForward
	v := e.ActiveView()
	c := v.Cursor()
	ok := c.NextWord()
	if !ok {
		e.SetStatus("End of buffer")
		return
	}
	v.MoveCursorTo(c)
}

type normalMode struct {
	editor *Editor
	count  string
}

func newNormalMode(e *Editor) *normalMode {
	m := normalMode{editor: e}
	m.editor.SetStatus("Normal")
	return &m
}

func (m *normalMode) onKey(ev *termbox.Event) {
	// Most of the key bindings are derived from those at
	// http://elvis.the-little-red-haired-girl.org/elvisman/elvisvi.html#index

	g := m.editor
	v := g.active.leaf

	// Consequtive non-zero digits specify action multiplier;
	// accumulate and return. Accept zero only if it's
	// a non-starting character.
	if ('0' < ev.Ch && ev.Ch <= '9') || (ev.Ch == '0' && len(m.count) > 0) {
		m.count = m.count + string(ev.Ch)
		m.editor.SetStatus(m.count)
		return
	}

	count := parseCount(m.count)

	var command Command

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
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
		case termbox.KeyCtrlD:
			// TODO: should move by count lines, default to 1/2 screen
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		case termbox.KeyCtrlE:
			// TODO: should move by count lines, default to 1
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		case termbox.KeyCtrlF:
			//TODO: should move a full page
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		case termbox.KeyCtrlG:
			v.onVcommand(viewCommand{Cmd: vCommandDisplayFileStatus})
		case termbox.KeyCtrlH:
			// Same as 'h'
			// TODO: find a way to avoid duplication of 'h'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBackward, Count: count})
		case termbox.KeyCtrlJ, termbox.KeyCtrlN:
			// Same as 'j'
			// TODO: find a way to avoid duplication of 'j'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorNextLine, Count: count})
		case termbox.KeyCtrlL:
			// TODO: redraw screen
			return
		case termbox.KeyCtrlM:
			// TODO: move to front of next line
			return
		case termbox.KeyCtrlP:
			// same as 'k'
			// TODO: find a way to avoid duplication of 'k'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorPrevLine, Count: count})
		case termbox.KeyCtrlR:
			v.onVcommand(viewCommand{Cmd: vCommandRedo, Count: count})
		case termbox.KeyCtrlU:
			//TODO: should move by count lines, default to 1/2 screen
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
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
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
		case termbox.KeyEsc:
			//TODO: Cancel the current command
			return
		case termbox.KeySpace:
			// Same as 'l'
			// TODO: find a way to avoid duplication of 'l'
			v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward, Count: count})
		}
	case 'A':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfLine})
		g.setMode(newInsertMode(g, count))
	case 'B':
		// TODO: Distinction from 'b'
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordBackward, Count: count})
	case 'C':
		v.onVcommand(viewCommand{Cmd: vCommandDeleteToEndOfLine})
		g.setMode(newInsertMode(g, count))
	case 'D':
		v.onVcommand(viewCommand{Cmd: vCommandDeleteToEndOfLine})
	case 'E':
		// TODO: Distinction from 'e'
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordEnd, Count: count})
	case 'F':
		// TODO: Move left to given character
		return
	case 'G':
		// TODO: Move to line #, default last line
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfFile})
	case 'H':
		// TODO: Move to line at the top of the screen
		return
	case 'I':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorFrontOfLine})
		g.setMode(newInsertMode(g, count))
	case 'J':
		// TODO: Join lines, whitespace separated
		return
	case 'K':
		// TODO: Run keywordprog
		return
	case 'L':
		// TODO: Move to line at the bottom of the screen
		return
	case 'M':
		// TODO: Move to line in the middle of the screen
		return
	case 'N':
		// TODO: Repeat previous search, backwards
		return
	case 'O':
		// TODO: Open new line above current
		return
	case 'P':
		// TODO: Paste text before cursor
		return
	case 'Q':
		// TODO: Quit to ex mode
		return
	case 'R':
		// TODO: Replace mode
		return
	case 'S':
		// TODO: Like 'cc'
		return
	case 'T':
		// TODO: Move left to just before the given character
		return
	case 'W':
		// TODO: Make distinct from 'w'
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordForward, Count: count})
	case 'X':
		// TODO: Delete count character to left of cursor
		return
	case 'Y':
		// TODO: Yank lines
		return
	case 'h':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBackward, Count: count})
	case 'j':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorNextLine, Count: count})
	case 'k':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorPrevLine, Count: count})
	case 'l':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward, Count: count})
	case 'w':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordForward, Count: count})
	case 'e':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordEnd, Count: count})
	case 'b':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordBackward, Count: count})
	case 'x':
		v.onVcommand(viewCommand{Cmd: vCommandDeleteRune, Count: count})
	case 'u':
		v.onVcommand(viewCommand{Cmd: vCommandUndo, Count: count})
	}

	// Insert mode; record first, then repeat.
	switch ev.Ch {
	case 'a':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward})
		g.setMode(newInsertMode(g, count))
	case 'i':
		g.setMode(newInsertMode(g, count))
	}

	// No point repeating these commands
	switch ev.Ch {
	case '0':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBeginningOfLine})
	case '$':
		v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfLine})
	}

	switch ev.Ch {
	case 'd':
		g.setMode(newTextObjectMode(g, m, v.buf.DeleteRange, count))
	}

	if ev.Ch == 0x0 {
		switch ev.Key {
		// TODO Cursor centering after Ctrl-U/D seems off.
		// TODO Ctrl-U and CTRL-D have configurable ranges of motion.
		case termbox.KeyCtrlU, termbox.KeyCtrlB:
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
		case termbox.KeyCtrlD, termbox.KeyCtrlF:
			v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		}
	}

	// TODO use count to set range for command mode
	switch ev.Ch {
	case ':':
		g.setMode(newCommandMode(g, m))
	}

	if command != nil {
		m.editor.Commands <- command
	}

	// Reset repetitions
	m.count = ""
}

// Parse action multiplier from a string.
func parseCount(s string) int {
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
