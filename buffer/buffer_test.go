package buffer

import (
	"bytes"
	"strings"
	"testing"
)

func checkLines(t *testing.T, b *Buffer, lines ...*Line) {
	if b.NumLines != len(lines) {
		t.Errorf("bad number of lines: got %d, want %d", b.NumLines, len(lines))
	}

	bufferLine := b.FirstLine
	for i, line := range lines {
		if line != bufferLine {
			t.Errorf("%d: line does not match buffer line", i)
		}

		// Check Prev
		if i == 0 {
			if line.Prev != nil {
				t.Error("first line has non-nil prev")
			}
		} else {
			if line.Prev != lines[i-1] {
				t.Errorf("%d: bad connection to previous line", i)
			}
		}

		// Check Next
		if i == len(lines)-1 {
			if line.Next != nil {
				t.Error("last line has non-nil next")
			}
		} else {
			if line.Next != lines[i+1] {
				t.Errorf("%d: bad connection to next line", i)
			}
		}

		bufferLine = bufferLine.Next
	}
}

func checkLineBytes(t *testing.T, b *Buffer, expected [][]byte) {
	if b.NumLines != len(expected) {
		t.Errorf("Unexpected line count: %d != %d", b.NumLines, len(expected))
	}
	l := b.FirstLine
	for i, e := range expected {
		if bytes.Compare(l.Data, e) != 0 {
			t.Errorf("Line %d does't match: '%s' != '%s'", i, l.Data, e)
		}
		l = l.Next
	}
}

func TestInsertLine(t *testing.T) {
	b := NewEmptyBuffer()
	l1 := &Line{Data: []byte("abcd")}
	l2 := &Line{Data: []byte("cdef")}
	l3 := &Line{Data: []byte("ghij")}

	// Append line
	b.InsertLine(l1, b.FirstLine)
	checkLines(t, b, b.FirstLine, l1)

	// Insert line between 1 and 2
	b.InsertLine(l2, b.FirstLine)
	checkLines(t, b, b.FirstLine, l2, l1)

	// Insert a line before 1
	first := b.FirstLine
	b.InsertLine(l3, nil)
	checkLines(t, b, l3, first, l2, l1)
}

func TestDeleteFirstLine(t *testing.T) {
	b, err := NewBuffer(strings.NewReader("foo\nbar"))
	if err != nil {
		t.Error("Error creating buffer")
	}

	b.DeleteLine(b.FirstLine)
	if b.NumLines != 1 {
		t.Error("Wrong number of lines")
	}
	if b.LastLine != b.FirstLine {
		t.Error("Wrong last line")
	}
	if b.FirstLine.Prev != nil {
		t.Error("Wrong line connection")
	}
	if b.FirstLine.Next != nil {
		t.Error("Wrong line connection")
	}
}

func TestDeleteLastLine(t *testing.T) {
	b, err := NewBuffer(strings.NewReader("foo\nbar"))
	if err != nil {
		t.Error("Error creating buffer")
	}

	b.DeleteLine(b.LastLine)
	if b.NumLines != 1 {
		t.Error("Wrong number of lines")
	}
	if b.LastLine != b.FirstLine {
		t.Error("Wrong last line")
	}
	if b.FirstLine.Prev != nil {
		t.Error("Wrong line connection")
	}
	if b.FirstLine.Next != nil {
		t.Error("Wrong line connection")
	}
}

func TestDeleteMiddleLine(t *testing.T) {
	b, err := NewBuffer(strings.NewReader("foo\nbar\nbaz"))
	if err != nil {
		t.Error("Error creating buffer")
	}

	l1 := b.FirstLine
	l2 := b.FirstLine.Next
	l3 := b.LastLine

	b.DeleteLine(l2)

	if b.NumLines != 2 {
		t.Error("Wrong number of lines")
	}
	if b.FirstLine != l1 {
		t.Error("Wrong last line")
	}
	if b.LastLine != l3 {
		t.Error("Wrong last line")
	}
	if l1.Next != l3 || l3.Prev != l1 {
		t.Error("Wrong line connection")
	}
}

func TestCleanupTrailingSpaces(t *testing.T) {
	b, err := NewBuffer(strings.NewReader(" blah \nfoo\nbar   \n  baz\n"))
	if err != nil {
		t.Error("Error creating buffer")
	}
	b.CleanupTrailingSpaces()
	checkLineBytes(t, b, [][]byte{
		[]byte(" blah"),
		[]byte("foo"),
		[]byte("bar"),
		[]byte("  baz"),
		[]byte(""),
	})
}

func TestCleanupTrailingNewlines(t *testing.T) {
	b, err := NewBuffer(strings.NewReader("\nfoo\n\nbar\n\n"))
	if err != nil {
		t.Error("Error creating buffer")
	}
	b.CleanupTrailingNewlines()
	checkLineBytes(t, b, [][]byte{
		[]byte(""),
		[]byte("foo"),
		[]byte(""),
		[]byte("bar"),
		// One newline left at the end.
		[]byte(""),
	})
}

func TestEnsureTrailingEOL(t *testing.T) {
	b, err := NewBuffer(strings.NewReader("foo"))
	if err != nil {
		t.Error("Error creating buffer")
	}
	b.EnsureTrailingEOL()
	checkLineBytes(t, b, [][]byte{
		[]byte("foo"),
		[]byte(""),
	})
}
