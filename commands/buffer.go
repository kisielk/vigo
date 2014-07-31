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
		if t.Buf.Name == "unnamed" && t.Buf.Path == "" {
			t.Buf.Name = e.BufferName(t.Filename)
		}
	if t.Filename != "" {
		err = t.Buf.SaveAs(t.Filename)
		if err != nil {
			t.Buf.Name = "unnamed"  // reset the name... something went wrong
			e.SetStatus(err)
		}
	} else {
		err = t.Buf.Save()
		if err != nil {
			e.SetStatus(err)
		}
	}
}
