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
	line     *line
	line_num int
	boffset  int
}

func (c *cursor) rune_under() (rune, int) {
	return utf8.DecodeRune(c.line.data[c.boffset:])
}

func (c *cursor) rune_before() (rune, int) {
	return utf8.DecodeLastRune(c.line.data[:c.boffset])
}

func (c *cursor) first_line() bool {
	return c.line.prev == nil
}

func (c *cursor) last_line() bool {
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
	if b.line_num < a.line_num {
		a, b = b, a
		s = -1
	} else if a.line_num == b.line_num && b.boffset < a.boffset {
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
func (c *cursor) voffset_coffset() (vo, co int) {
	data := c.line.data[:c.boffset]
	for len(data) > 0 {
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		co += 1
		vo += rune_advance_len(r, vo)
	}
	return
}

// Find a visual offset for a given cursor
func (c *cursor) voffset() (vo int) {
	data := c.line.data[:c.boffset]
	for len(data) > 0 {
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		vo += rune_advance_len(r, vo)
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

func (c *cursor) extract_bytes(n int) []byte {
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

func (c *cursor) NextRune() {
	if c.eol() {
		return
	} else {
		_, rlen := c.rune_under()
		c.boffset += rlen
	}
}

func (c *cursor) PrevRune() {
	if c.bol() {
		return
	} else {
		_, rlen := c.rune_before()
		c.boffset -= rlen
	}
}

func (c *cursor) move_beginning_of_line() {
	c.boffset = 0
}

func (c *cursor) move_end_of_line() {
	c.boffset = len(c.line.data)
}

func (c *cursor) word_under_cursor() []byte {
	end, beg := *c, *c
	r, rlen := beg.rune_before()
	if r == utf8.RuneError {
		return nil
	}

	for IsWord(r) && !beg.bol() {
		beg.boffset -= rlen
		r, rlen = beg.rune_before()
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
			if c.last_line() {
				return false
			} else {
				c.line = c.line.next
				c.line_num++
				c.boffset = 0
				continue
			}
		}

		r, rlen := c.rune_under()
		for !f(r) && !c.eol() {
			c.boffset += rlen
			r, rlen = c.rune_under()
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
func (c *cursor) NextWord() bool {
	isNotSpace := func(r rune) bool {
		return !unicode.IsSpace(r)
	}
	r, _ := c.rune_under()
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
func (c *cursor) move_one_word_forward() bool {
	// move cursor forward until the first word rune is met
	for {
		if c.eol() {
			if c.last_line() {
				return false
			} else {
				c.line = c.line.next
				c.line_num++
				c.boffset = 0
				continue
			}
		}
		r, rlen := c.rune_under()
		for !IsWord(r) && !c.eol() {
			c.boffset += rlen
			r, rlen = c.rune_under()
		}
		if c.eol() {
			continue
		}
		break
	}
	// now the cursor is under the word rune, skip all of them
	r, rlen := c.rune_under()
	for IsWord(r) && !c.eol() {
		c.boffset += rlen
		r, rlen = c.rune_under()
	}
	return true
}

// Move cursor backward until current rune satisfies condition f.
// Returns true if the move was successful, false if EOF reached.
func (c *cursor) prevRuneFunc(f func(rune) bool) bool {
	for {
		if c.bol() {
			if c.first_line() {
				return false
			} else {
				c.line = c.line.prev
				c.line_num--
				c.boffset = len(c.line.data)
				continue
			}
		}
		r, rlen := c.rune_before()
		for !f(r) && !c.bol() {
			c.boffset -= rlen
			r, rlen = c.rune_before()
		}
		break
	}
	return true
}

// Move cursor forward to beginning of the previous word.
// Skips the rest of the current word, if any, unless is located at its
// first character. Returns true if the move was successful, false if EOF reached.
func (c *cursor) PrevWord() bool {
	isNotSpace := func(r rune) bool {
		return !unicode.IsSpace(r)
	}
	// Skip remaining whitespace until next word of any type.
	_ = c.prevRuneFunc(isNotSpace)
	r, _ := c.rune_before()
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
func (c *cursor) move_one_word_backward() bool {
	// move cursor backward while previous rune is not a word rune
	for {
		if c.bol() {
			if c.first_line() {
				return false
			} else {
				c.line = c.line.prev
				c.line_num--
				c.boffset = len(c.line.data)
				continue
			}
		}

		r, rlen := c.rune_before()
		for !IsWord(r) && !c.bol() {
			c.boffset -= rlen
			r, rlen = c.rune_before()
		}

		if c.bol() {
			continue
		}
		break
	}

	// now the rune behind the cursor is a word rune, while it's true, move
	// backwards
	r, rlen := c.rune_before()
	for IsWord(r) && !c.bol() {
		c.boffset -= rlen
		r, rlen = c.rune_before()
	}

	return true
}

func (c *cursor) on_insert_adjust(a *action) {
	if a.cursor.line_num > c.line_num {
		return
	}
	if a.cursor.line_num < c.line_num {
		// inserted something above the cursor, adjust it
		c.line_num += len(a.lines)
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
			c.line = a.last_line()
			c.line_num += len(a.lines)
			c.boffset = a.last_line_affection_len() +
				c.boffset - a.cursor.boffset
		}
	}
}

func (c *cursor) on_delete_adjust(a *action) {
	if a.cursor.line_num > c.line_num {
		return
	}
	if a.cursor.line_num < c.line_num {
		// deletion above the cursor line, may touch the cursor location
		if len(a.lines) == 0 {
			// no lines were deleted, no things to adjust
			return
		}

		first, last := a.deleted_lines()
		if first <= c.line_num && c.line_num <= last {
			// deleted the cursor line, see how much it affects it
			n := 0
			if last == c.line_num {
				n = c.boffset - a.last_line_affection_len()
				if n < 0 {
					n = 0
				}
			}
			*c = a.cursor
			c.boffset += n
		} else {
			// phew.. no worries
			c.line_num -= len(a.lines)
			return
		}
	}

	// the last case is deletion on the cursor line, see what was deleted
	if a.cursor.boffset >= c.boffset {
		// deleted something after cursor, don't care
		return
	}

	n := c.boffset - (a.cursor.boffset + a.first_line_affection_len())
	if n < 0 {
		n = 0
	}
	c.boffset = a.cursor.boffset + n
}

func (c cursor) search_forward(word []byte) (cursor, bool) {
	for c.line != nil {
		i := bytes.Index(c.line.data[c.boffset:], word)
		if i != -1 {
			c.boffset += i
			return c, true
		}

		c.line = c.line.next
		c.line_num++
		c.boffset = 0
	}
	return c, false
}

func (c cursor) search_backward(word []byte) (cursor, bool) {
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
		c.line_num--
		c.boffset = len(c.line.data)
	}
	return c, false
}

func swap_cursors_maybe(c1, c2 cursor) (r1, r2 cursor) {
	if c1.line_num == c2.line_num {
		if c1.boffset > c2.boffset {
			return c2, c1
		} else {
			return c1, c2
		}
	}

	if c1.line_num > c2.line_num {
		return c2, c1
	}
	return c1, c2
}
