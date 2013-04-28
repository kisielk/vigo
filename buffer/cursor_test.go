package buffer

import (
	"testing"
)

// makeLines converts text into an array of lines.
func makeLines(text ...string) []*Line {
	lines := []*Line{}
	current := (*Line)(nil)
	for i := 0; i < len(text); i++ {
		next := &Line{Data: []byte(text[i]), Prev: current}
		if current != nil {
			current.Next = next
		}
		current = next
		lines = append(lines, next)
	}
	return lines
}

func TestNextRune(t *testing.T) {
	lines := makeLines(
		"// comment",
		"func bar(i int) {",
	)
	l0 := lines[0]

	// Start of line 1
	c := &Cursor{Line: l0, Boffset: 0}

	// Go forward one character at a time
	for i := 1; i < len(l0.Data); i++ {
		c.NextRune(false)
		if c.Line != l0 {
			t.Error("Bad cursor line at index", i)
		}
		if c.Boffset != i {
			t.Error("Bad cursor offset at index", i)
		}
	}

	// Cursor should stay at the end of the line
	for i := 0; i < 3; i++ {
		c.NextRune(false)
		if c.Line != l0 {
			t.Error("Bad cursor line")
		}
		if c.Boffset != len(l0.Data) {
			t.Error("Bad cursor index")
		}
	}
}

func TestNextRuneWrap(t *testing.T) {
	lines := makeLines(
		"// comment",
		"func bar(i int) {",
	)

	// End of line 1
	c := &Cursor{Line: lines[0], Boffset: 9}

	// FIXME currently cursors go to EOL which is one past the last
	// character; for now, needs an extra motion to wrap to next line.
	c.NextRune(true)
	c.NextRune(true)
	if c.Line != lines[1] {
		t.Error("Cursor did not wrap to next line")
	}
	if c.Boffset != 0 {
		t.Error("Cursor wrapped to wrong offset", c.Boffset)
	}
}

func TestPrevRune(t *testing.T) {
	lines := makeLines(
		"// comment",
		"func bar(i int) {",
	)
	l0 := lines[0]

	// End of line 1
	c := &Cursor{Line: l0, Boffset: 9}

	// Go backwards one character at a time
	for i := len(l0.Data) - 2; i >= 0; i-- {
		c.PrevRune(false)
		if c.Line != l0 {
			t.Error("Bad cursor line at index", i)
		}
		if c.Boffset != i {
			t.Error("Bad cursor offset at index", i)
		}
	}

	// Cursor should stay at the beginning of the line
	for i := 0; i < 3; i++ {
		c.PrevRune(false)
		if c.Line != l0 {
			t.Error("Bad cursor line")
		}
		if c.Boffset != 0 {
			t.Error("Bad cursor index")
		}
	}
}

func TestPrevRuneWrap(t *testing.T) {
	lines := makeLines(
		"// comment",
		"func bar(i int) {",
	)

	// Beginning of line 2
	c := &Cursor{Line: lines[1], Boffset: 0}

	c.PrevRune(true)
	if c.Line != lines[0] {
		t.Error("Cursor did not wrap to previous line")
	}
	if c.Boffset != 10 {
		t.Error("Cursor wrapped to wrong offset", c.Boffset)
	}
}

func TestNextWord(t *testing.T) {
	// TODO test EOF, test empty line
	lines := makeLines(
		"// comment",
		"func bar(i int) {",
		" return 0",
		"}",
	)
	stops := []Cursor{
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
	c := &Cursor{Line: lines[1], Boffset: 2}

	for i := 0; i < len(stops); i++ {
		c.NextWord()
		s := stops[i]
		if c.Line != s.Line {
			t.Error("Bad cursor line at index", i, c.Line, "!=", s.Line)
		}
		if c.Boffset != s.Boffset {
			t.Error("Bad cursor position at index", i, c.Boffset, "!=", s.Boffset)
		}
	}
}

func TestPrevWord(t *testing.T) {
	// TODO test BOF, test empty line
	lines := makeLines(
		"// comment",
		"func bar(i int) {",
		" return 0",
		"}",
	)
	stops := []Cursor{
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
	c := &Cursor{Line: lines[2], Boffset: 8}

	for i := 0; i < len(stops); i++ {
		c.PrevWord()
		s := stops[i]
		if c.Line != s.Line {
			t.Error("Bad cursor line at index", i)
		}
		if c.Boffset != s.Boffset {
			t.Error("Bad cursor position at index", i)
		}
	}
}

func TestPrevWordSpaces(t *testing.T) {
	// Skipping words backward on line with leading spaces
	lines := makeLines(
		"  foo",
		"  bar",
	)
	// Second line, beginning of 'bar'
	c := &Cursor{Line: lines[1], Boffset: 2}

	// Should jump to be beginning of foo
	c.PrevWord()
	if c.Line != lines[0] {
		t.Error("Bad cursor line")
	}
	if c.Boffset != 2 {
		t.Error("Bad cursor position", c.Boffset)
	}
}

func TestSortCursors(t *testing.T) {

	c1 := Cursor{nil, 1, 10}
	c2 := Cursor{nil, 1, 20}
	c3 := Cursor{nil, 2, 10}

	var pairs = []struct {
		in1  Cursor
		in2  Cursor
		out1 Cursor
		out2 Cursor
	}{
		{c1, c2, c1, c2},
		{c2, c1, c1, c2},
		{c1, c3, c1, c3},
		{c3, c1, c1, c3},
	}

	for _, p := range pairs {
		out1, out2 := SortCursors(p.in1, p.in2)
		if out1 != p.out1 || out2 != p.out2 {
			t.Error("Wrong cursor order")
		}
	}
}
