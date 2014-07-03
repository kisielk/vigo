package commands

import (
	"github.com/kisielk/vigo/editor"
	"github.com/kisielk/vigo/buffer"
)

type SaveBuffer struct {
	Buf *buffer.Buffer
	Filename string
}

func (t SaveBuffer) Apply(e *editor.Editor) {
	var err error
	if t.Filename != "" {
		err = t.Buf.SaveAs(t.Filename)
		if err != nil {
			e.SetStatus(err)
		}
		if t.Buf.Name == "" {
			t.Buf.Name = e.BufferName(t.Filename)
		}
	}
}
