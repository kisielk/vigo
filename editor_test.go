package main

import "testing"

func TestValidCutBuffer(t *testing.T) {
	// 1-9 are the anonymous buffers
	for i := byte('1'); i <= '9'; i++ {
		validCutBuffer(i)
	}
	// a-z are the named buffers
	for i := byte('a'); i <= 'z'; i++ {
		validCutBuffer(i)
	}
	// special buffer .
	validCutBuffer('.')

	// Ensure that any other buffer names panic
	invalidCutBuffer := func(b byte) {
		defer func() {
			err := recover()
			if err == nil {
				t.Fail()
				t.Logf("no panic for %q", b)
			}
		}()
		validCutBuffer(b)
	}
	for i := byte(0); i < '1'; i++ {
		if i == '.' {
			continue
		}
		invalidCutBuffer(i)
	}
	for i := byte('9') + 1; i < 'a'; i++ {
		invalidCutBuffer(i)
	}
	for i := byte('z') + 1; i != 0; i++ {
		invalidCutBuffer(i)
	}
}

func TestNamedCutBuffers(t *testing.T) {
	bufs := NewCutBuffers()
	for i := byte('a'); i <= 'z'; i++ {
		if b := bufs.Get(i); len(b) != 0 {
			t.Logf("%s: unexpected length: %d", i, len(b))
			t.Fail()
		}
	}

	in := "foobar"
	for i := byte('a'); i <= 'z'; i++ {
		bufs.Set(i, []byte(in+string(i)))
	}
	for i := byte('a'); i <= 'z'; i++ {
		expected := in + string(i)
		if out := bufs.Get(i); string(out) != expected {
			t.Logf("%s: got %q, want %q", i, out, expected)
			t.Fail()
		}
	}
}

func TestAnonCutBuffers(t *testing.T) {
	bufs := NewCutBuffers()
	for i := byte('1'); i <= '9'; i++ {
		if b := bufs.Get(i); len(b) != 0 {
			t.Logf("%s: unexpected length: %d", i, len(b))
			t.Fail()
		}
	}

	in := "hello"
	for i := byte('9'); i >= '1'; i-- {
		bufs.UpdateAnon([]byte(in + string(i)))
	}
	for i := byte('1'); i <= '9'; i++ {
		expected := in + string(i)
		if out := bufs.Get(i); string(out) != expected {
			t.Logf("%s: got %q, want %q", string(i), out, expected)
			t.Fail()
		}
	}
}
