package mode

import (
	cmd "github.com/kisielk/vigo/commands"
	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/utils"
	"github.com/nsf/termbox-go"
)

type NormalMode struct {
	editor  *editor.Editor
	count   string
	command editor.Command
}

func NewNormalMode(e *editor.Editor) *NormalMode {
	m := NormalMode{editor: e}
	m.editor.SetStatus("Normal")
	return &m
}

func (m *NormalMode) Enter(e *editor.Editor) {
	e.ActiveView().Buffer().FinalizeActionGroup()
}

func (m *NormalMode) Reset() {
	m.count = ""
	m.command = nil
}

func (m *NormalMode) OnKey(ev *termbox.Event) {
	// Most of the key bindings are derived from those at
	// http://elvis.the-little-red-haired-girl.org/elvisman/elvisvi.html#index

	g := m.editor
	v := g.ActiveView()
	c := v.Cursor()

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

	// Consecutive comands that require futher input to complete;
	// accumulate and return.  Accept escape to reset the command
	switch ev.Ch {
	case 0x0:
		switch ev.Key {
		case termbox.KeyEsc:
      m.Reset()
      return
		}
	}

	// TODO: For (half)screen moving commands, use view.Height() in
	// future cleanup. Currently, that method is private.
	viewHeight := g.Height() - 1

	switch ev.Ch {
	case 0x0:
		// TODO Cursor centering after Ctrl-U/D seems off.
		// TODO Ctrl-U and CTRL-D have configurable ranges of motion.
		switch ev.Key {
		case termbox.KeyCtrlA:
			term := c.WordUnderCursor()
			if term != nil {
				storeSearchTerm(g, string(term))
				g.Commands <- cmd.Search{Dir: cmd.Forward}
			}
		case termbox.KeyCtrlB:
			g.Commands <- cmd.MoveView{Dir: cmd.Backward, Lines: viewHeight}
		case termbox.KeyCtrlD:
			// TODO: should move by count lines, default to 1/2 screen
			g.Commands <- cmd.MoveView{Dir: cmd.Forward, Lines: viewHeight / 2}
		case termbox.KeyCtrlE:
			// TODO: should move by count lines, default to 1
			g.Commands <- cmd.MoveView{Dir: cmd.Forward, Lines: 1}
		case termbox.KeyCtrlF:
			g.Commands <- cmd.MoveView{Dir: cmd.Forward, Lines: viewHeight}
		case termbox.KeyCtrlG:
			g.Commands <- cmd.DisplayFileStatus{}
		case termbox.KeyCtrlH:
			// Same as 'h'
			g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward, Wrap: false}, count}
		case termbox.KeyCtrlJ, termbox.KeyCtrlN:
			// Same as 'j'
			g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
		case termbox.KeyCtrlL:
			// TODO: redraw screen
			return
		case termbox.KeyCtrlM:
			g.Commands <- cmd.MoveLine{Dir: cmd.Forward}
			g.Commands <- cmd.MoveFOL{}
		case termbox.KeyCtrlP:
			// same as 'k'
			g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
		case termbox.KeyCtrlR:
			g.Commands <- cmd.Repeat{cmd.Redo{}, count}
		case termbox.KeyCtrlU:
			// TODO: should move by count lines, default to 1/2 screen
			g.Commands <- cmd.MoveView{Dir: cmd.Backward, Lines: viewHeight / 2}
		case termbox.KeyCtrlV:
			// TODO: Start visual (block) selection
			return
		case termbox.KeyCtrlW:
			// TODO Use count for window width/height
			g.SetMode(NewWindowMode(g, count))
		case termbox.KeyCtrlX:
			// TODO: Move to column count
			return
		case termbox.KeyCtrlY:
			// TODO: should move by count lines, default to 1
			g.Commands <- cmd.MoveView{Dir: cmd.Backward, Lines: 1}
		case termbox.KeySpace:
			// Same as 'l'
			g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Forward, Wrap: false}, count}
		}
	case 'A':
		g.Commands <- cmd.MoveEOL{}
		g.SetMode(NewInsertMode(g, count))
	case 'B':
		// TODO: Distinction from 'b'
		g.Commands <- cmd.Repeat{cmd.MoveWord{Dir: cmd.Backward}, count}
	case 'C':
		g.Commands <- cmd.DeleteEOL{}
		g.SetMode(NewInsertMode(g, count))
	case 'D':
		g.Commands <- cmd.DeleteEOL{}
	case 'E':
		// TODO: Distinction from 'e'
		g.Commands <- cmd.Repeat{cmd.MoveWordEnd{}, count}
	case 'F':
		// TODO: Distinction from 'f' - move the the first (nth) occurence of following rune
		return
	case 'G':
		// TODO: Move to line #, default last line
		g.Commands <- cmd.MoveEOF{}
	case 'H':
		// TODO: Move to line at the top of the screen
		return
	case 'I':
		g.Commands <- cmd.MoveFOL{}
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
		g.Commands <- cmd.Search{Dir: cmd.Backward}
	case 'O':
		g.Commands <- cmd.Repeat{cmd.NewLine{Dir: cmd.Backward}, count}
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
		g.Commands <- cmd.Repeat{cmd.MoveWord{Dir: cmd.Forward}, count}
	case 'X':
		// TODO: Delete count character to left of cursor
		return
	case 'Y':
		// TODO: Yank lines
		return
	case '0':
		g.Commands <- cmd.MoveBOL{}
	case '$':
		g.Commands <- cmd.MoveEOL{}
	case '^':
		g.Commands <- cmd.MoveFOL{}
	case 'h':
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Backward, Wrap: false}, count}
	case 'j':
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Forward}, count}
	case 'k':
		g.Commands <- cmd.Repeat{cmd.MoveLine{Dir: cmd.Backward}, count}
	case 'l':
		g.Commands <- cmd.Repeat{cmd.MoveRune{Dir: cmd.Forward, Wrap: false}, count}
	case 'o':
		g.Commands <- cmd.Repeat{cmd.NewLine{Dir: cmd.Forward}, count}
		g.SetMode(NewInsertMode(g, count))
	case 'w':
		g.Commands <- cmd.Repeat{cmd.MoveWord{Dir: cmd.Forward}, count}
	case 'e':
		g.Commands <- cmd.Repeat{cmd.MoveWordEnd{}, count}
	case 'f':
		// TODO: Move to first (nth) occurance of entered rune to the right
	case 'b':
		g.Commands <- cmd.Repeat{cmd.MoveWord{Dir: cmd.Backward}, count}
	case 'x':
		g.Commands <- cmd.Repeat{cmd.DeleteRune{}, count}
	case 'u':
		g.Commands <- cmd.Repeat{cmd.Undo{}, count}
	case 'n':
		g.Commands <- cmd.Search{Dir: cmd.Forward}
	}

	switch ev.Ch {
	case 'a':
		g.Commands <- cmd.MoveRune{Dir: cmd.Forward, Wrap: false}
		g.SetMode(NewInsertMode(g, count))
	case 'd':
		g.SetMode(NewTextObjectMode(g, m, v.Buffer().DeleteRange, count))
	case 'i':
		g.SetMode(NewInsertMode(g, count))
	case 'v':
		g.SetMode(NewVisualMode(g, false))
	case 'V':
		g.SetMode(NewVisualMode(g, false))
	case ':':
		// TODO use count to set range for command mode
		g.SetMode(NewCommandMode(g, m))
	case '/':
		g.SetMode(NewSearchMode(g, m))
	}

	// Reset repetitions
  m.Reset()
}

func (m *NormalMode) Exit() {
}
