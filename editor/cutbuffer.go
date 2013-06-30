package editor

import (
	"fmt"
)

type cutBuffers map[byte][]byte

func newCutBuffers() *cutBuffers {
	c := make(cutBuffers, 36)
	return &c
}

// UpdateAnon the contents of the anonymous cut buffer 1
// with the given byte slice s, and rotates the rest of the buffers
func (bs *cutBuffers) updateAnon(s []byte) {
	bufs := *bs
	for i := byte('9'); i > '1'; i-- {
		bufs[i] = bufs[i-1]
	}
	bufs['1'] = s
	*bs = bufs
}

// validCutBuffer panics if b is not a valid cut buffer name
// b must a character between a-z, 1-9, or .
func validCutBuffer(b byte) {
	if b != '.' && b < '1' || b > '9' && b < 'a' || b > 'z' {
		panic(fmt.Errorf("invalid cut buffer: %q", b))
	}
}

// Set updates the contents of the cut buffer b with the byte slice s
func (bs *cutBuffers) set(b byte, s []byte) {
	validCutBuffer(b)
	(*bs)[b] = s
}

// Append appends the byte slice s to the contents of buffer b
func (bs *cutBuffers) append(b byte, s []byte) {
	validCutBuffer(b)
	(*bs)[b] = append((*bs)[b], s...)
}

// Get returns the contents of the cut buffer b
func (bs *cutBuffers) get(b byte) []byte {
	validCutBuffer(b)
	return (*bs)[b]
}
