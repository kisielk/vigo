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
