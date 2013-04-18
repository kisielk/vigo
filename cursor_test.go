package main

import (
	"testing"
)

func TestNextWord(t *testing.T) {

	a := new(line)
	a.data = []byte("func bar(i int) int")
	// TODO test word motion between two lines.

	c := &cursor{line: a, boffset: 0}

	// Expected cursor byte offsets
	stops := []int{5, 8, 9, 11, 14, 16}

	for i := 0; i < len(stops); i++ {
		c.NextWord()
		if c.boffset != stops[i] {
			t.Error("Bad cursor position at index", i)
		}
	}
}
