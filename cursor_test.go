package main

import (
	"testing"
)

// Returns new copy of test data.
func makeLines() []*line {

	// Create test data
	text := [][]byte{
		[]byte("// comment"),
		[]byte("func bar(i int) {"),
		[]byte(" return 0"),
		[]byte("}"),
	}

	lines := [4]*line{}

	first := &line{data: text[0]}
	current := first
	lines[0] = first

	for i := 1; i < len(text); i++ {
		next := &line{data: text[i], prev: current}
		current.next = next
		current = next
		lines[i] = next
	}

	return lines[:]
}

func TestNextRune(t *testing.T) {
	lines := makeLines()
	l0 := lines[0]

	// Start of line 1
	c := &cursor{line: l0, boffset: 0}

	// Go forward one character at a time
	for i := 1; i < len(l0.data); i++ {
		c.NextRune(false)
		if c.line != l0 {
			t.Error("Bad cursor line at index", i)
		}
		if c.boffset != i {
			t.Error("Bad cursor offset at index", i)
		}
	}

	// Cursor should stay at the end of the line
	for i := 0; i < 3; i++ {
		c.NextRune(false)
		if c.line != l0 {
			t.Error("Bad cursor line")
		}
		if c.boffset != len(l0.data) {
			t.Error("Bad cursor index")
		}
	}
}

func TestNextRuneWrap(t *testing.T) {
	lines := makeLines()

	// End of line 1
	c := &cursor{line: lines[0], boffset: 9}

	// FIXME currently cursors go to EOL which is one past the last
	// character; for now, needs an extra motion to wrap to next line.
	c.NextRune(true)
	c.NextRune(true)
	if c.line != lines[1] {
		t.Error("Cursor did not wrap to next line")
	}
	if c.boffset != 0 {
		t.Error("Cursor wrapped to wrong offset", c.boffset)
	}
}

func TestPrevRune(t *testing.T) {
	lines := makeLines()
	l0 := lines[0]

	// End of line 1
	c := &cursor{line: l0, boffset: 9}

	// Go backwards one character at a time
	for i := len(l0.data) - 2; i >= 0; i-- {
		c.PrevRune(false)
		if c.line != l0 {
			t.Error("Bad cursor line at index", i)
		}
		if c.boffset != i {
			t.Error("Bad cursor offset at index", i)
		}
	}

	// Cursor should stay at the beginning of the line
	for i := 0; i < 3; i++ {
		c.PrevRune(false)
		if c.line != l0 {
			t.Error("Bad cursor line")
		}
		if c.boffset != 0 {
			t.Error("Bad cursor index")
		}
	}
}

func TestPrevRuneWrap(t *testing.T) {
	lines := makeLines()

	// Beginning of line 2
	c := &cursor{line: lines[1], boffset: 0}

	c.PrevRune(true)
	if c.line != lines[0] {
		t.Error("Cursor did not wrap to previous line")
	}
	if c.boffset != 10 {
		t.Error("Cursor wrapped to wrong offset", c.boffset)
	}
}

func TestNextWord(t *testing.T) {
	// TODO test EOF, test empty line
	lines := makeLines()
	stops := []cursor{
		{lines[1], 1, 5},
		{lines[1], 1, 8},
		{lines[1], 1, 9},
		{lines[1], 1, 11},
		{lines[1], 1, 14},
		{lines[1], 1, 16},
		{lines[2], 2, 1},
		{lines[2], 2, 8},
		{lines[3], 3, 0},
	}

	// Start at line 2 offset 2 (func)
	c := &cursor{line: lines[1], boffset: 2}

	for i := 0; i < len(stops); i++ {
		c.NextWord()
		s := stops[i]
		if c.line != s.line {
			t.Error("Bad cursor line at index", i, c.line, "!=", s.line)
		}
		if c.boffset != s.boffset {
			t.Error("Bad cursor position at index", i, c.boffset, "!=", s.boffset)
		}
	}
}

func TestPrevWord(t *testing.T) {
	// TODO test BOF, test empty line
	lines := makeLines()
	stops := []cursor{
		{lines[2], 2, 1},
		{lines[1], 1, 16},
		{lines[1], 1, 14},
		{lines[1], 1, 11},
		{lines[1], 1, 9},
		{lines[1], 1, 8},
		{lines[1], 1, 5},
		{lines[1], 1, 0},
		{lines[0], 0, 3},
		{lines[0], 0, 0},
	}

	// Position at the end of line 3
	c := &cursor{line: lines[2], boffset: 8}

	for i := 0; i < len(stops); i++ {
		c.PrevWord()
		s := stops[i]
		if c.line != s.line {
			t.Error("Bad cursor line at index", i)
		}
		if c.boffset != s.boffset {
			t.Error("Bad cursor position at index", i)
		}
	}
}
