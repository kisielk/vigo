package editor

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"unicode/utf8"
)

//----------------------------------------------------------------------------
// line
//----------------------------------------------------------------------------

type line struct {
	data []byte
	next *line
	prev *line
}

// Find a set of closest offsets for a given visual offset
func (l *line) findClosestOffsets(voffset int) (bo, co, vo int) {
	data := l.data
	for len(data) > 0 {
		var vodif int
		r, rlen := utf8.DecodeRune(data)
		data = data[rlen:]
		vodif = runeAdvanceLen(r, vo)
		if vo+vodif > voffset {
			return
		}

		bo += rlen
		co += 1
		vo += vodif
	}
	return
}

//----------------------------------------------------------------------------
// buffer
//----------------------------------------------------------------------------

type buffer struct {
	views     []*view
	firstLine *line
	lastLine  *line
	loc       viewLocation
	linesN    int
	bytesN    int
	history   *actionGroup
	onDisk    *actionGroup
	mark      cursor

	// absoulte path of the file, if it's empty string, then the file has no
	// on-disk representation
	path string

	// buffer name (displayed in the status line), must be unique,
	// uniqueness is maintained by godit methods
	name string
}

func newEmptyBuffer() *buffer {
	b := new(buffer)
	l := new(line)
	l.next = nil
	l.prev = nil
	b.firstLine = l
	b.lastLine = l
	b.linesN = 1
	b.loc = viewLocation{
		topLine:    l,
		topLineNum: 1,
		cursor: cursor{
			line:    l,
			lineNum: 1,
		},
	}
	b.initHistory()
	return b
}

func newBuffer(r io.Reader) (*buffer, error) {
	var err error
	var prevline *line

	br := bufio.NewReader(r)
	l := new(line)
	b := new(buffer)
	b.loc = viewLocation{
		topLine:    l,
		topLineNum: 1,
		cursor: cursor{
			line:    l,
			lineNum: 1,
		},
	}
	b.linesN = 1
	b.firstLine = l
	for {
		l.data, err = br.ReadBytes('\n')
		if err != nil {
			// last line was read
			break
		} else {
			b.bytesN += len(l.data)

			// cut off the '\n' character
			l.data = l.data[:len(l.data)-1]
		}

		b.linesN++
		l.next = new(line)
		l.prev = prevline
		prevline = l
		l = l.next
	}
	l.prev = prevline
	b.lastLine = l

	// io.EOF is not an error
	if err == io.EOF {
		err = nil
	}

	// history
	b.initHistory()
	return b, err
}

func (b *buffer) addView(v *view) {
	b.views = append(b.views, v)
}

func (b *buffer) deleteView(v *view) {
	vi := -1
	for i, n := 0, len(b.views); i < n; i++ {
		if b.views[i] == v {
			vi = i
			break
		}
	}

	if vi != -1 {
		lasti := len(b.views) - 1
		b.views[vi], b.views[lasti] = b.views[lasti], b.views[vi]
		b.views = b.views[:lasti]
	}
}

func (b *buffer) otherViews(v *view, cb func(*view)) {
	for _, ov := range b.views {
		if v == ov {
			continue
		}
		cb(ov)
	}
}

func (b *buffer) initHistory() {
	// the trick here is that I set 'sentinel' as 'history', it is required
	// to maintain an invariant, where 'history' is a sentinel or is not
	// empty

	sentinel := new(actionGroup)
	first := new(actionGroup)
	sentinel.next = first
	first.prev = sentinel
	b.history = sentinel
	b.onDisk = sentinel
}

func (b *buffer) isMarkSet() bool {
	return b.mark.line != nil
}

func (b *buffer) dumpHistory() {
	cur := b.history
	for cur.prev != nil {
		cur = cur.prev
	}

	p := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}

	i := 0
	for cur != nil {
		p("action group %d: %d actions\n", i, len(cur.actions))
		for _, a := range cur.actions {
			switch a.what {
			case actionInsert:
				p(" + insert")
			case actionDelete:
				p(" - delete")
			}
			p(" (%2d,%2d):%q\n", a.cursor.lineNum,
				a.cursor.boffset, string(a.data))
		}
		cur = cur.next
		i++
	}
}

func (b *buffer) save() error {
	return b.saveAs(b.path)
}

func (b *buffer) saveAs(filename string) error {
	r := b.reader()
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if err != nil {
		return err
	}

	b.onDisk = b.history
	for _, v := range b.views {
		v.dirty |= DIRTY_STATUS
	}
	return nil
}

func (b *buffer) syncedWithDisk() bool {
	return b.onDisk == b.history
}

func (b *buffer) reader() *bufferReader {
	return newBufferReader(b)
}

func (b *buffer) contents() []byte {
	data, _ := ioutil.ReadAll(b.reader())
	return data
}

//----------------------------------------------------------------------------
// buffer_reader
//----------------------------------------------------------------------------

type bufferReader struct {
	buffer *buffer
	line   *line
	offset int
}

func newBufferReader(buffer *buffer) *bufferReader {
	br := new(bufferReader)
	br.buffer = buffer
	br.line = buffer.firstLine
	br.offset = 0
	return br
}

func (br *bufferReader) Read(data []byte) (int, error) {
	nread := 0
	for len(data) > 0 {
		if br.line == nil {
			return nread, io.EOF
		}

		// how much can we read from current line
		canRead := len(br.line.data) - br.offset
		if len(data) <= canRead {
			// if this is all we need, return
			n := copy(data, br.line.data[br.offset:])
			nread += n
			br.offset += n
			break
		}

		// otherwise try to read '\n' and jump to the next line
		n := copy(data, br.line.data[br.offset:])
		nread += n
		data = data[n:]
		if len(data) > 0 && br.line != br.buffer.lastLine {
			data[0] = '\n'
			data = data[1:]
			nread++
		}

		br.line = br.line.next
		br.offset = 0
	}
	return nread, nil
}
