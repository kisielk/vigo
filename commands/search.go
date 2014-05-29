package commands

import (
	"bytes"
	"github.com/kisielk/vigo/editor"
)

type Search struct {
	Dir  Dir
}

func (s Search) Apply(e *editor.Editor) {
	v := e.ActiveView()
	c := v.Cursor()

	if e.LastSearchTerm == "" {
		e.SetStatus("Nothing to search for.")
		return
	}
	word := []byte(e.LastSearchTerm)

	switch s.Dir {
	case Forward:
		e.SetStatus("Search forward for: %s", e.LastSearchTerm)
		for c.Line != nil {

			// move the cursor one run forward.
			// this allows us to move to the next match.
			// without this, if the word under the cursor is a match,
			// then we won't be able to advance to the next match
			c.NextRune(false)

			i := bytes.Index(c.Line.Data[c.Boffset:], word)
			if i != -1 {
				c.Boffset += i
				break
			}

			c.Line = c.Line.Next
			if c.Line == nil {
				e.SetStatus("No more results")
				return
			}

			c.LineNum++
			c.Boffset = 0
		}
	case Backward:
		e.SetStatus("Search backward for: %s", e.LastSearchTerm)
		for {
			i := bytes.LastIndex(c.Line.Data[:c.Boffset], word)

			if i != -1 {
				c.Boffset = i
				break
			}

			c.Line = c.Line.Prev
			if c.Line == nil {
				e.SetStatus("No previous results")
				return
			}
			c.LineNum--
			c.Boffset = len(c.Line.Data)
		}
	}

	v.MoveCursorTo(c)
}


func searchForward(e *editor.Editor) {
	
}
