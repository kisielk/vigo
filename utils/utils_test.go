package utils

import "bytes"
import "testing"

func TestIterWords(t *testing.T) {
	tests := []struct {
		in  []byte
		out [][]byte
	}{
		{[]byte("hello world"), bytes.Split([]byte("hello:world"), []byte(":"))},
		{[]byte("    hello    world   "), bytes.Split([]byte("hello:world"), []byte(":"))},
	}

	for i, test := range tests {
		out := [][]byte{}
		f := func(word []byte) {
			out = append(out, word)
		}
		IterWords(test.in, f)
		if len(out) != len(test.out) {
			t.Logf("%d: wrong output length: got %d want %d", i, len(out), len(test.out))
		}
		for j := range out {
			if !bytes.Equal(out[j], test.out[j]) {
				t.Logf("%d:%d: don't match. got %q want %q", i, j, out[j], test.out[j])
				t.Fail()
			}
		}
	}
}
