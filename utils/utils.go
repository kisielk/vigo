package utils

import (
	"bytes"
	"strconv"
	"unicode"
)

// TODO keep synched with editor/config.go/tabstopLength
const TabstopLength = 8

var InvisibleRuneTable = []rune{
	'@',  // 0
	'A',  // 1
	'B',  // 2
	'C',  // 3
	'D',  // 4
	'E',  // 5
	'F',  // 6
	'G',  // 7
	'H',  // 8
	'I',  // 9
	'J',  // 10
	'K',  // 11
	'L',  // 12
	'M',  // 13
	'N',  // 14
	'O',  // 15
	'P',  // 16
	'Q',  // 17
	'R',  // 18
	'S',  // 19
	'T',  // 20
	'U',  // 21
	'V',  // 22
	'W',  // 23
	'X',  // 24
	'Y',  // 25
	'Z',  // 26
	'[',  // 27
	'\\', // 28
	']',  // 29
	'^',  // 30
	'_',  // 31
}

func IterWords(data []byte, cb func(word []byte)) {
	for {
		i := bytes.IndexFunc(data, IsWord)
		if i == -1 {
			return
		}
		data = data[i:]
		i = bytes.IndexFunc(data, func(r rune) bool {
			return !IsWord(r)
		})
		if i == -1 {
			return
		}
		cb(data[:i])
		data = data[i:]
	}
}

func IndexFirstNonSpace(s []byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] != '\t' && s[i] != ' ' {
			return i
		}
	}
	return len(s)
}

func IndexLastNonSpace(s []byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != '\t' && s[i] != ' ' {
			return i
		}
	}
	return -1
}

func CloneByteSlice(s []byte) []byte {
	c := make([]byte, len(s))
	copy(c, s)
	return c
}

func IsWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// Function will iterate 'data' contents, calling 'cb' on some data or on '\n',
// but never both. For example, given this data: "\n123\n123\n\n", it will call
// 'cb' 6 times: ['\n', '123', '\n', '123', '\n', '\n']
func IterLines(data []byte, cb func([]byte)) {
	offset := 0
	for {
		if offset == len(data) {
			return
		}

		i := bytes.IndexByte(data[offset:], '\n')
		switch i {
		case -1:
			cb(data[offset:])
			return
		case 0:
			cb(data[offset : offset+1])
			offset++
			continue
		}

		cb(data[offset : offset+i])
		cb(data[offset+i : offset+i+1])
		offset += i + 1
	}
}

func InsertBytes(s []byte, offset int, data []byte) []byte {
	n := len(s) + len(data)
	s = GrowByteSlice(s, n)
	s = s[:n]
	copy(s[offset+len(data):], s[offset:])
	copy(s[offset:], data)
	return s
}

func RuneAdvanceLen(r rune, pos int) int {
	switch {
	case r == '\t':
		return TabstopLength - pos%TabstopLength
	case r < 32:
		// for invisible chars like ^R ^@ and such, two cells
		return 2
	}
	return 1
}

func GrowByteSlice(s []byte, desiredCap int) []byte {
	if cap(s) < desiredCap {
		ns := make([]byte, len(s), desiredCap)
		copy(ns, s)
		return ns
	}
	return s
}

// ParseCount parses action multiplier from a string.
func ParseCount(s string) int {
	var n int64 = 1
	var err error
	if len(s) > 0 {
		n, err = strconv.ParseInt(s, 10, 32)
		if err != nil {
			panic("could not parse action multiplier")
		}
	}
	return int(n)
}
