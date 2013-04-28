package buffer

import (
	"bytes"
	"strings"
	"testing"
)

func checkAction(t *testing.T, a *Action, ref *Action) {
	if a.What != ref.What {
		t.Error("wrong type")
	}
	if bytes.Compare(a.Data, ref.Data) != 0 {
		t.Errorf("wrong data")
	}
	if a.Cursor != ref.Cursor {
		t.Error("wrong cursor")
	}
	if len(a.Lines) != len(ref.Lines) {
		t.Errorf("wrong number of lines")
	}
	for i, la := range a.Lines {
		lref := ref.Lines[i]
		// TODO add CompareLine?
		if bytes.Compare(la.Data, lref.Data) != 0 {
			t.Errorf("%d: wrong line data", i)
		}
	}
}

func TestNewInsertAction(t *testing.T) {
	l := &Line{Data: []byte("abcd")}
	c := Cursor{
		Line:    l,
		LineNum: 0,
		Boffset: 3,
	}
	a := NewInsertAction(c, []byte("hjkl"))
	ref := &Action{
		What:   ActionInsert,
		Data:   []byte("hjkl"),
		Cursor: c,
		Lines:  []*Line{},
	}
	checkAction(t, a, ref)
}

func TestNewInsertActionMultiline(t *testing.T) {
	l := &Line{Data: []byte("abcd")}
	c := Cursor{
		Line:    l,
		LineNum: 0,
		Boffset: 3,
	}
	a := NewInsertAction(c, []byte("hj\nkl"))
	ref := &Action{
		What:   ActionInsert,
		Data:   []byte("hj\nkl"),
		Cursor: c,
		Lines:  []*Line{&Line{}},
	}
	checkAction(t, a, ref)
}

func TestNewDeleteAction(t *testing.T) {
	l := &Line{Data: []byte("abcde fgh")}
	c := Cursor{
		Line:    l,
		LineNum: 0,
		Boffset: 3,
	}
	a := NewDeleteAction(c, 4)
	ref := &Action{
		What:   ActionDelete,
		Data:   []byte("de f"),
		Cursor: c,
		Lines:  []*Line{},
	}
	checkAction(t, a, ref)
}

func TestNewDeleteActionMultiline(t *testing.T) {
	b, err := NewBuffer(strings.NewReader("abc\ndef"))
	if err != nil {
		t.Error("Error creating buffer")
	}
	c := Cursor{
		Line:    b.FirstLine,
		LineNum: 0,
		Boffset: 1,
	}
	a := NewDeleteAction(c, 4)
	ref := &Action{
		What:   ActionDelete,
		Data:   []byte("bc\nd"),
		Cursor: c,
		Lines:  []*Line{b.LastLine},
	}
	checkAction(t, a, ref)
}
