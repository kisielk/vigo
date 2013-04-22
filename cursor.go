package main

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

//----------------------------------------------------------------------------
// cursor location
//----------------------------------------------------------------------------

type cursor struct {
	line    *line
	lineNum int
	boffset int
}

func (c *cursor) runeUnder() (rune, int) {
	return utf8.DecodeRune(c.line.data[c.boffset:])
}

func (c *cursor) runeBefore() (rune, int) {
	return utf8.DecodeLastRune(c.line.data[:c.boffset])
}

func (c *cursor) firstLine() bool {
	return c.line.prev == nil
}

func (c *cursor) lastLine() bool {
	return c.line.next == nil
}

// end of line
func (c *cursor) eol() bool {
	return c.boffset == len(c.line.data)
}

// beginning of line
func (c *cursor) bol() bool {
	return c.boffset == 0
}

// returns the distance between two locations in bytes
func (a cursor) distance(b cursor) int {
	s := 1
	if b.lineNum < a.lineNum {
		a, b = b, a
		s = -1
	} else if a.lineNum == b.lineNum && b.boffset < a.boffset {
		a, b = b, a
		s = -1
	}

	n := 0
	for a.line != b.line {
		n += len(a.line.data) - a.boffset + 1
		a.line = a.line.next
		a.boffset = 0
	}
	n += b.boffset - a.boffset
	return n * s
}

// Find a visual and a character offset for a given cursor
func (c *cursor) voffsetCoffset() (vo, co int) {
	data := c.line.data[:c.boffset]
	for len(data) > 0 {
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		co += 1
		vo += runeAdvanceLen(r, vo)
	}
	return
}

// Find a visual offset for a given cursor
func (c *cursor) voffset() (vo int) {
	data := c.line.data[:c.boffset]
	for len(data) > 0 {
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		vo += runeAdvanceLen(r, vo)
	}
	return
}

func (c *cursor) coffset() (co int) {
	data := c.line.data[:c.boffset]
	for len(data) > 0 {
		_, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		co += 1
	}
	return
}

func (c *cursor) extractBytes(n int) []byte {
	var buf bytes.Buffer
	offset := c.boffset
	line := c.line
	for n > 0 {
		switch {
		case offset < len(line.data):
			nb := len(line.data) - offset
			if n < nb {
				nb = n
			}
			buf.Write(line.data[offset : offset+nb])
			n -= nb
			offset += nb
		case offset == len(line.data):
			buf.WriteByte('\n')
			offset = 0
			line = line.next
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
func (c *cursor) nextRune(wrap bool) bool {
	if !c.eol() {
		_, rlen := c.runeUnder()
		c.boffset += rlen
		return true
	} else if wrap && !c.lastLine() {
		c.line = c.line.next
		c.lineNum++
		c.boffset = 0
		return true
	}
	return false
}

// PrevRune moves cursor to the previous rune. If wrap is true,
// wraps the cursor to the end of next line once the beginning of
// the current one is reached. Returns true if motion succeeded,
// false otherwise.
func (c *cursor) prevRune(wrap bool) bool {
	if !c.bol() {
		_, rlen := c.runeBefore()
		c.boffset -= rlen
		return true
	} else if wrap && !c.firstLine() {
		c.line = c.line.prev
		c.lineNum--
		c.boffset = len(c.line.data)
		return true
	}
	return false
}

func (c *cursor) moveBeginningOfLine() {
	c.boffset = 0
}

func (c *cursor) moveEndOfLine() {
	c.boffset = len(c.line.data)
}

func (c *cursor) wordUnderCursor() []byte {
	end, beg := *c, *c
	r, rlen := beg.runeBefore()
	if r == utf8.RuneError {
		return nil
	}

	for IsWord(r) && !beg.bol() {
		beg.boffset -= rlen
		r, rlen = beg.runeBefore()
	}

	if beg.boffset == end.boffset {
		return nil
	}
	return c.line.data[beg.boffset:end.boffset]
}

// Move cursor forward until current rune satisfies condition f.
// Returns true if the move was successful, false if EOF reached.
func (c *cursor) nextRuneFunc(f func(rune) bool) bool {
	for {

		if c.eol() {
			if c.lastLine() {
				return false
			} else {
				c.line = c.line.next
				c.lineNum++
				c.boffset = 0
				continue
			}
		}

		r, rlen := c.runeUnder()
		for !f(r) && !c.eol() {
			c.boffset += rlen
			r, rlen = c.runeUnder()
		}

		if c.eol() {
			continue
		}

		break
	}

	return true
}

// Move cursor forward to beginning of next word.
// Skips the rest of the current word, if any. Returns true if
// the move was successful, false if EOF reached.
func (c *cursor) nextWord() bool {
	isNotSpace := func(r rune) bool {
		return !unicode.IsSpace(r)
	}
	r, _ := c.runeUnder()
	if isNotSpace(r) {
		// Lowercase word motion differentiates words consisting of
		// (A-Z0-9_) and any other non-whitespace character. Skip until
		// we find either the other word type or whitespace.
		if IsWord(r) {
			c.nextRuneFunc(func(r rune) bool {
				return !IsWord(r) || unicode.IsSpace(r)
			})
		} else {
			c.nextRuneFunc(func(r rune) bool {
				return IsWord(r) || unicode.IsSpace(r)
			})
		}
	}
	// Skip remaining whitespace until next word of any type.
	return c.nextRuneFunc(isNotSpace)
}

// returns true if the move was successful, false if EOF reached.
func (c *cursor) moveOneWordForward() bool {
	// move cursor forward until the first word rune is met
	for {
		if c.eol() {
			if c.lastLine() {
				return false
			} else {
				c.line = c.line.next
				c.lineNum++
				c.boffset = 0
				continue
			}
		}
		r, rlen := c.runeUnder()
		for !IsWord(r) && !c.eol() {
			c.boffset += rlen
			r, rlen = c.runeUnder()
		}
		if c.eol() {
			continue
		}
		break
	}
	// now the cursor is under the word rune, skip all of them
	r, rlen := c.runeUnder()
	for IsWord(r) && !c.eol() {
		c.boffset += rlen
		r, rlen = c.runeUnder()
	}
	return true
}

// Move cursor backward until current rune satisfies condition f.
// Returns true if the move was successful, false if EOF reached.
func (c *cursor) prevRuneFunc(f func(rune) bool) bool {
	for {
		if c.bol() {
			if c.firstLine() {
				return false
			} else {
				c.line = c.line.prev
				c.lineNum--
				c.boffset = len(c.line.data)
				continue
			}
		}
		r, rlen := c.runeBefore()
		for !f(r) && !c.bol() {
			c.boffset -= rlen
			r, rlen = c.runeBefore()
		}
		break
	}
	return true
}

// Move cursor forward to beginning of the previous word.
// Skips the rest of the current word, if any, unless is located at its
// first character. Returns true if the move was successful, false if EOF reached.
func (c *cursor) prevWord() bool {
	isNotSpace := func(r rune) bool {
		return !unicode.IsSpace(r)
	}
	// Skip remaining whitespace until next word of any type.
	_ = c.prevRuneFunc(isNotSpace)
	r, _ := c.runeBefore()
	if isNotSpace(r) {
		// Lowercase word motion differentiates words consisting of
		// (A-Z0-9_) and any other non-whitespace character. Skip until
		// we find either the other word type or whitespace.
		if IsWord(r) {
			c.prevRuneFunc(func(r rune) bool {
				return !IsWord(r) || unicode.IsSpace(r)
			})
		} else {
			c.prevRuneFunc(func(r rune) bool {
				return IsWord(r) || unicode.IsSpace(r)
			})
		}
	}
	return !c.bol()
}

// returns true if the move was successful, false if BOF reached.
func (c *cursor) moveOneWordBackward() bool {
	// move cursor backward while previous rune is not a word rune
	for {
		if c.bol() {
			if c.firstLine() {
				return false
			} else {
				c.line = c.line.prev
				c.lineNum--
				c.boffset = len(c.line.data)
				continue
			}
		}

		r, rlen := c.runeBefore()
		for !IsWord(r) && !c.bol() {
			c.boffset -= rlen
			r, rlen = c.runeBefore()
		}

		if c.bol() {
			continue
		}
		break
	}

	// now the rune behind the cursor is a word rune, while it's true, move
	// backwards
	r, rlen := c.runeBefore()
	for IsWord(r) && !c.bol() {
		c.boffset -= rlen
		r, rlen = c.runeBefore()
	}

	return true
}

func (c *cursor) onInsertAdjust(a *action) {
	if a.cursor.lineNum > c.lineNum {
		return
	}
	if a.cursor.lineNum < c.lineNum {
		// inserted something above the cursor, adjust it
		c.lineNum += len(a.lines)
		return
	}

	// insertion on the cursor line
	if a.cursor.boffset < c.boffset {
		// insertion before the cursor, move cursor along with insertion
		if len(a.lines) == 0 {
			// no lines were inserted, simply adjust the offset
			c.boffset += len(a.data)
		} else {
			// one or more lines were inserted, adjust cursor
			// respectively
			c.line = a.lastLine()
			c.lineNum += len(a.lines)
			c.boffset = a.lastLineAffectionLen() +
				c.boffset - a.cursor.boffset
		}
	}
}

func (c *cursor) onDeleteAdjust(a *action) {
	if a.cursor.lineNum > c.lineNum {
		return
	}
	if a.cursor.lineNum < c.lineNum {
		// deletion above the cursor line, may touch the cursor location
		if len(a.lines) == 0 {
			// no lines were deleted, no things to adjust
			return
		}

		first, last := a.deletedLines()
		if first <= c.lineNum && c.lineNum <= last {
			// deleted the cursor line, see how much it affects it
			n := 0
			if last == c.lineNum {
				n = c.boffset - a.lastLineAffectionLen()
				if n < 0 {
					n = 0
				}
			}
			*c = a.cursor
			c.boffset += n
		} else {
			// phew.. no worries
			c.lineNum -= len(a.lines)
			return
		}
	}

	// the last case is deletion on the cursor line, see what was deleted
	if a.cursor.boffset >= c.boffset {
		// deleted something after cursor, don't care
		return
	}

	n := c.boffset - (a.cursor.boffset + a.firstLineAffectionLen())
	if n < 0 {
		n = 0
	}
	c.boffset = a.cursor.boffset + n
}

func (c cursor) searchForward(word []byte) (cursor, bool) {
	for c.line != nil {
		i := bytes.Index(c.line.data[c.boffset:], word)
		if i != -1 {
			c.boffset += i
			return c, true
		}

		c.line = c.line.next
		c.lineNum++
		c.boffset = 0
	}
	return c, false
}

func (c cursor) searchBackward(word []byte) (cursor, bool) {
	for {
		i := bytes.LastIndex(c.line.data[:c.boffset], word)
		if i != -1 {
			c.boffset = i
			return c, true
		}

		c.line = c.line.prev
		if c.line == nil {
			break
		}
		c.lineNum--
		c.boffset = len(c.line.data)
	}
	return c, false
}

func swapCursorMaybe(c1, c2 cursor) (r1, r2 cursor) {
	if c1.lineNum == c2.lineNum {
		if c1.boffset > c2.boffset {
			return c2, c1
		} else {
			return c1, c2
		}
	}

	if c1.lineNum > c2.lineNum {
		return c2, c1
	}
	return c1, c2
}
