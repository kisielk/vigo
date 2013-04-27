package buffer

import (
	"bytes"
	"github.com/kisielk/vigo/utils"
	"unicode"
	"unicode/utf8"
)

type Cursor struct {
	Line    *Line
	LineNum int
	Boffset int
}

func (c *Cursor) RuneUnder() (rune, int) {
	return utf8.DecodeRune(c.Line.Data[c.Boffset:])
}

func (c *Cursor) RuneBefore() (rune, int) {
	return utf8.DecodeLastRune(c.Line.Data[:c.Boffset])
}

func (c *Cursor) FirstLine() bool {
	return c.Line.Prev == nil
}

func (c *Cursor) LastLine() bool {
	return c.Line.Next == nil
}

// end of line
func (c *Cursor) EOL() bool {
	return c.Boffset == len(c.Line.Data)
}

// beginning of line
func (c *Cursor) BOL() bool {
	return c.Boffset == 0
}

// returns the distance between two locations in bytes
func (a Cursor) distance(b Cursor) int {
	s := 1
	if b.LineNum < a.LineNum {
		a, b = b, a
		s = -1
	} else if a.LineNum == b.LineNum && b.Boffset < a.Boffset {
		a, b = b, a
		s = -1
	}

	n := 0
	for a.Line != b.Line {
		n += len(a.Line.Data) - a.Boffset + 1
		a.Line = a.Line.Next
		a.Boffset = 0
	}
	n += b.Boffset - a.Boffset
	return n * s
}

// Find a visual and a character offset for a given cursor
func (c *Cursor) voffsetCoffset() (vo, co int) {
	data := c.Line.Data[:c.Boffset]
	for len(data) > 0 {
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		co += 1
		vo += utils.RuneAdvanceLen(r, vo)
	}
	return
}

// Find a visual offset for a given cursor
func (c *Cursor) VisualOffset() int {
	vo, _ := c.voffsetCoffset()
	return vo
}

func (c *Cursor) coffset() (co int) {
	data := c.Line.Data[:c.Boffset]
	for len(data) > 0 {
		_, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		co += 1
	}
	return
}

func (c *Cursor) extractBytes(n int) []byte {
	var buf bytes.Buffer
	offset := c.Boffset
	line := c.Line
	for n > 0 {
		switch {
		case offset < len(line.Data):
			nb := len(line.Data) - offset
			if n < nb {
				nb = n
			}
			buf.Write(line.Data[offset : offset+nb])
			n -= nb
			offset += nb
		case offset == len(line.Data):
			buf.WriteByte('\n')
			offset = 0
			line = line.Next
			n -= 1
		default:
			panic("unreachable")
		}
	}
	return buf.Bytes()
}

// NextRune moves cursor to the next rune. If wrap is true,
// wraps the cursor to the beginning of next line once the end
// of the current one is reached. Returns true if motion succeeded,
// false otherwise.
func (c *Cursor) NextRune(wrap bool) bool {
	if !c.EOL() {
		_, rlen := c.RuneUnder()
		c.Boffset += rlen
		return true
	} else if wrap && !c.LastLine() {
		c.Line = c.Line.Next
		c.LineNum++
		c.Boffset = 0
		return true
	}
	return false
}

// PrevRune moves cursor to the previous rune. If wrap is true,
// wraps the cursor to the end of next line once the beginning of
// the current one is reached. Returns true if motion succeeded,
// false otherwise.
func (c *Cursor) PrevRune(wrap bool) bool {
	if !c.BOL() {
		_, rlen := c.RuneBefore()
		c.Boffset -= rlen
		return true
	} else if wrap && !c.FirstLine() {
		c.Line = c.Line.Prev
		c.LineNum--
		c.Boffset = len(c.Line.Data)
		return true
	}
	return false
}

func (c *Cursor) MoveBOL() {
	c.Boffset = 0
}

func (c *Cursor) MoveEOL() {
	c.Boffset = len(c.Line.Data)
}

func (c *Cursor) WordUnderCursor() []byte {
	end, beg := *c, *c
	r, rlen := beg.RuneBefore()
	if r == utf8.RuneError {
		return nil
	}

	for utils.IsWord(r) && !beg.BOL() {
		beg.Boffset -= rlen
		r, rlen = beg.RuneBefore()
	}

	if beg.Boffset == end.Boffset {
		return nil
	}
	return c.Line.Data[beg.Boffset:end.Boffset]
}

// Move cursor forward until current rune satisfies condition f.
// Returns true if the move was successful, false if EOF reached.
func (c *Cursor) NextRuneFunc(f func(rune) bool) bool {
	for {
		if c.EOL() {
			if c.LastLine() {
				return false
			} else {
				c.Line = c.Line.Next
				c.LineNum++
				c.Boffset = 0
				continue
			}
		}
		r, rlen := c.RuneUnder()
		for !f(r) && !c.EOL() {
			c.Boffset += rlen
			r, rlen = c.RuneUnder()
		}
		if c.EOL() {
			continue
		}
		break
	}
	return true
}

// Move cursor forward to beginning of next word.
// Skips the rest of the current word, if any. Returns true if
// the move was successful, false if EOF reached.
func (c *Cursor) NextWord() bool {
	isNotSpace := func(r rune) bool {
		return !unicode.IsSpace(r)
	}
	r, _ := c.RuneUnder()
	if isNotSpace(r) {
		// Lowercase word motion differentiates words consisting of
		// (A-Z0-9_) and any other non-whitespace character. Skip until
		// we find either the other word type or whitespace.
		if utils.IsWord(r) {
			c.NextRuneFunc(func(r rune) bool {
				return !utils.IsWord(r) || unicode.IsSpace(r)
			})
		} else {
			c.NextRuneFunc(func(r rune) bool {
				return utils.IsWord(r) || unicode.IsSpace(r)
			})
		}
	}
	// Skip remaining whitespace until next word of any type.
	return c.NextRuneFunc(isNotSpace)
}

// returns true if the move was successful, false if EOF reached.
func (c *Cursor) MoveOneWordForward() bool {
	// move cursor forward until the first word rune is met
	for {
		if c.EOL() {
			if c.LastLine() {
				return false
			} else {
				c.Line = c.Line.Next
				c.LineNum++
				c.Boffset = 0
				continue
			}
		}
		r, rlen := c.RuneUnder()
		for !utils.IsWord(r) && !c.EOL() {
			c.Boffset += rlen
			r, rlen = c.RuneUnder()
		}
		if c.EOL() {
			continue
		}
		break
	}
	// now the cursor is under the word rune, skip all of them
	r, rlen := c.RuneUnder()
	for utils.IsWord(r) && !c.EOL() {
		c.Boffset += rlen
		r, rlen = c.RuneUnder()
	}
	return true
}

// Move cursor backward until current rune satisfies condition f.
// Returns true if the move was successful, false if EOF reached.
func (c *Cursor) PrevRuneFunc(f func(rune) bool) bool {
	for {
		if c.BOL() {
			if c.FirstLine() {
				return false
			} else {
				c.Line = c.Line.Prev
				c.LineNum--
				c.Boffset = len(c.Line.Data)
				continue
			}
		}
		r, rlen := c.RuneBefore()
		for !f(r) && !c.BOL() {
			c.Boffset -= rlen
			r, rlen = c.RuneBefore()
		}
		break
	}
	return true
}

// Move cursor forward to beginning of the previous word.
// Skips the rest of the current word, if any, unless is located at its
// first character. Returns true if the move was successful, false if EOF reached.
func (c *Cursor) prevWord() bool {
	isNotSpace := func(r rune) bool {
		return !unicode.IsSpace(r)
	}
	for {
		// Skip space until we find a word character.
		// Re-try if we reached beginning-of-line.
		if !c.PrevRuneFunc(isNotSpace) {
			return false
		}
		if !c.BOL() {
			break
		}
	}
	r, _ := c.RuneBefore()
	if isNotSpace(r) {
		// Lowercase word motion differentiates words consisting of
		// (A-Z0-9_) and any other non-whitespace character. Skip until
		// we find either the other word type or whitespace.
		if utils.IsWord(r) {
			c.PrevRuneFunc(func(r rune) bool {
				return !utils.IsWord(r) || unicode.IsSpace(r)
			})
		} else {
			c.PrevRuneFunc(func(r rune) bool {
				return utils.IsWord(r) || unicode.IsSpace(r)
			})
		}
	}
	return !c.BOL()
}

// returns true if the move was successful, false if BOF reached.
func (c *Cursor) MoveOneWordBackward() bool {
	// move cursor backward while previous rune is not a word rune
	for {
		if c.BOL() {
			if c.FirstLine() {
				return false
			} else {
				c.Line = c.Line.Prev
				c.LineNum--
				c.Boffset = len(c.Line.Data)
				continue
			}
		}

		r, rlen := c.RuneBefore()
		for !utils.IsWord(r) && !c.BOL() {
			c.Boffset -= rlen
			r, rlen = c.RuneBefore()
		}
		if c.BOL() {
			continue
		}
		break
	}

	// now the rune behind the cursor is a word rune, while it's true, move
	// backwards
	r, rlen := c.RuneBefore()
	for utils.IsWord(r) && !c.BOL() {
		c.Boffset -= rlen
		r, rlen = c.RuneBefore()
	}

	return true
}

func (c *Cursor) onInsertAdjust(a *Action) {
	if a.cursor.LineNum > c.LineNum {
		return
	}
	if a.cursor.LineNum < c.LineNum {
		// inserted something above the cursor, adjust it
		c.LineNum += len(a.lines)
		return
	}

	// insertion on the cursor line
	if a.cursor.Boffset < c.Boffset {
		// insertion before the cursor, move cursor along with insertion
		if len(a.lines) == 0 {
			// no lines were inserted, simply adjust the offset
			c.Boffset += len(a.Data)
		} else {
			// one or more lines were inserted, adjust cursor
			// respectively
			c.Line = a.LastLine()
			c.LineNum += len(a.lines)
			c.Boffset = a.lastLineAffectionLen() +
				c.Boffset - a.cursor.Boffset
		}
	}
}

func (c *Cursor) onDeleteAdjust(a *Action) {
	if a.cursor.LineNum > c.LineNum {
		return
	}
	if a.cursor.LineNum < c.LineNum {
		// deletion above the cursor line, may touch the cursor location
		if len(a.lines) == 0 {
			// no lines were deleted, no things to adjust
			return
		}

		first, last := a.deletedLines()
		if first <= c.LineNum && c.LineNum <= last {
			// deleted the cursor line, see how much it affects it
			n := 0
			if last == c.LineNum {
				n = c.Boffset - a.lastLineAffectionLen()
				if n < 0 {
					n = 0
				}
			}
			*c = a.cursor
			c.Boffset += n
		} else {
			// phew.. no worries
			c.LineNum -= len(a.lines)
			return
		}
	}

	// the last case is deletion on the cursor line, see what was deleted
	if a.cursor.Boffset >= c.Boffset {
		// deleted something after cursor, don't care
		return
	}

	n := c.Boffset - (a.cursor.Boffset + a.firstLineAffectionLen())
	if n < 0 {
		n = 0
	}
	c.Boffset = a.cursor.Boffset + n
}

func (c Cursor) searchForward(word []byte) (Cursor, bool) {
	for c.Line != nil {
		i := bytes.Index(c.Line.Data[c.Boffset:], word)
		if i != -1 {
			c.Boffset += i
			return c, true
		}

		c.Line = c.Line.Next
		c.LineNum++
		c.Boffset = 0
	}
	return c, false
}

func (c Cursor) searchBackward(word []byte) (Cursor, bool) {
	for {
		i := bytes.LastIndex(c.Line.Data[:c.Boffset], word)
		if i != -1 {
			c.Boffset = i
			return c, true
		}

		c.Line = c.Line.Prev
		if c.Line == nil {
			break
		}
		c.LineNum--
		c.Boffset = len(c.Line.Data)
	}
	return c, false
}

func swapCursorMaybe(c1, c2 Cursor) (r1, r2 Cursor) {
	if c1.LineNum == c2.LineNum {
		if c1.Boffset > c2.Boffset {
			return c2, c1
		} else {
			return c1, c2
		}
	}

	if c1.LineNum > c2.LineNum {
		return c2, c1
	}
	return c1, c2
}
