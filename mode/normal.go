package mode

import (
	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/utils"
	"github.com/nsf/termbox-go"
)

type MoveWord struct {
}

func (m MoveWord) Apply(e *editor.Editor) {
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
	editor *editor.Editor
	count  string
}

func NewNormalMode(e *editor.Editor) *normalMode {
	m := normalMode{editor: e}
	m.editor.SetStatus("Normal")
	return &m
}

func (m *normalMode) repeat(cmd editor.Command, count int) {
	for i := 0; i < count; i++ {
		m.editor.Commands <- cmd
	}
}

func (m *normalMode) OnKey(ev *termbox.Event) {
	// Most of the key bindings are derived from those at
	// http://elvis.the-little-red-haired-girl.org/elvisman/elvisvi.html#index

	g := m.editor
	v := g.ActiveView()

	// Consequtive non-zero digits specify action multiplier;
	// accumulate and return. Accept zero only if it's
	// a non-starting character.
	if ('0' < ev.Ch && ev.Ch <= '9') || (ev.Ch == '0' && len(m.count) > 0) {
		m.count = m.count + string(ev.Ch)
		m.editor.SetStatus(m.count)
		return
	}

	count := utils.ParseCount(m.count)
	if count == 0 {
		count = 1
	}

	// var command editor.Command

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
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
		case termbox.KeyCtrlD:
			// TODO: should move by count lines, default to 1/2 screen
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		case termbox.KeyCtrlE:
			// TODO: should move by count lines, default to 1
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		case termbox.KeyCtrlF:
			//TODO: should move a full page
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		case termbox.KeyCtrlG:
			// v.onVcommand(viewCommand{Cmd: vCommandDisplayFileStatus})
		case termbox.KeyCtrlH:
			// Same as 'h'
			// TODO: find a way to avoid duplication of 'h'
			// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBackward, Count: count})
		case termbox.KeyCtrlJ, termbox.KeyCtrlN:
			// Same as 'j'
			// TODO: find a way to avoid duplication of 'j'
			// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorNextLine, Count: count})
		case termbox.KeyCtrlL:
			// TODO: redraw screen
			return
		case termbox.KeyCtrlM:
			// TODO: move to front of next line
			return
		case termbox.KeyCtrlP:
			// same as 'k'
			// TODO: find a way to avoid duplication of 'k'
			// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorPrevLine, Count: count})
		case termbox.KeyCtrlR:
			g.Commands <- cmd.Repeat{cmd.Redo{}, count}
		case termbox.KeyCtrlU:
			//TODO: should move by count lines, default to 1/2 screen
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
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
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
		case termbox.KeyEsc:
			//TODO: Cancel the current command
			return
		case termbox.KeySpace:
			// Same as 'l'
			// TODO: find a way to avoid duplication of 'l'
			// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward, Count: count})
		}
	case 'A':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfLine})
		g.SetMode(NewInsertMode(g, count))
	case 'B':
		// TODO: Distinction from 'b'
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordBackward, Count: count})
	case 'C':
		// v.onVcommand(viewCommand{Cmd: vCommandDeleteToEndOfLine})
		g.SetMode(NewInsertMode(g, count))
	case 'D':
		// v.onVcommand(viewCommand{Cmd: vCommandDeleteToEndOfLine})
	case 'E':
		// TODO: Distinction from 'e'
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordEnd, Count: count})
	case 'F':
		// TODO: Move left to given character
		return
	case 'G':
		// TODO: Move to line #, default last line
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfFile})
	case 'H':
		// TODO: Move to line at the top of the screen
		return
	case 'I':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorFrontOfLine})
		g.SetMode(NewInsertMode(g, count))
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
		// v.onVcommand(viewCommand{Cmd: vCommandNewLineAbove})
		g.SetMode(NewInsertMode(g, count))
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
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordForward, Count: count})
	case 'X':
		// TODO: Delete count character to left of cursor
		return
	case 'Y':
		// TODO: Yank lines
		return
	case 'h':
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward, Wrap: false}, count}
	case 'j':
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
	case 'k':
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
	case 'l':
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Forward, Wrap: false}, count}
	case 'o':
		// v.onVcommand(viewCommand{Cmd: vCommandNewLineBelow})
		g.SetMode(NewInsertMode(g, count))
	case 'w':
		g.Commands <- cmd.Repeat{cmd.MoveWord{Dir: cmd.Forward}, count}
	case 'e':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorWordEnd, Count: count})
	case 'b':
		g.Commands <- cmd.Repeat{cmd.MoveWord{Dir: cmd.Backward}, count}
	case 'x':
		g.Commands <- cmd.Repeat{cmd.DeleteRune{}, count}
	case 'u':
		g.Commands <- cmd.Repeat{cmd.Undo{}, count}
	}

	// Insert mode; record first, then repeat.
	switch ev.Ch {
	case 'a':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorForward})
		g.SetMode(NewInsertMode(g, count))
	case 'i':
		g.SetMode(NewInsertMode(g, count))
	}

	// No point repeating these commands
	switch ev.Ch {
	case '0':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorBeginningOfLine})
	case '$':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorEndOfLine})
	case '^':
		// v.onVcommand(viewCommand{Cmd: vCommandMoveCursorFrontOfLine})
	}

	switch ev.Ch {
	case 'd':
		g.SetMode(NewTextObjectMode(g, m, v.Buffer().DeleteRange, count))
	}

	if ev.Ch == 0x0 {
		switch ev.Key {
		// TODO Cursor centering after Ctrl-U/D seems off.
		// TODO Ctrl-U and CTRL-D have configurable ranges of motion.
		case termbox.KeyCtrlU, termbox.KeyCtrlB:
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfBackward, Count: count})
		case termbox.KeyCtrlD, termbox.KeyCtrlF:
			// v.onVcommand(viewCommand{Cmd: vCommandMoveViewHalfForward, Count: count})
		}
	}

	// TODO use count to set range for command mode
	switch ev.Ch {
	case ':':
		g.SetMode(editor.NewCommandMode(g, m))
	}

	// if command != nil {
	// 	m.editor.Commands <- command
	// }

	// Reset repetitions
	m.count = ""
}

func (m *normalMode) Exit() {
}
