package buffer

import (
	"strings"
	"testing"
)

func TestInsertLine(t *testing.T) {
	b := NewEmptyBuffer()
	l1 := &Line{Data: []byte("abcd")}
	l2 := &Line{Data: []byte("cdef")}

	// Append line
	b.InsertLine(l1, b.FirstLine)
	if b.NumLines != 2 {
		t.Error("Wrong number of lines")
	}
	if b.FirstLine.Next != l1 || l1.Prev != b.FirstLine {
		t.Error("Wrong line connection")
	}
	if b.LastLine != l1 {
		t.Error("Wrong last line")
	}

	// Insert line between 1 and 2
	b.InsertLine(l2, b.FirstLine)
	if b.FirstLine.Next != l2 || l2.Prev != b.FirstLine {
		t.Error("Wrong line connection")
	}
	if l2.Next != l1 || l1.Prev != l2 {
		t.Error("Wrong line connection")
	}
	if b.LastLine != l1 {
		t.Error("Wrong last line")
	}
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
