package commands

import (
	"github.com/kisielk/vigo/editor"
)

type DisplayFileStatus struct{}

func (r DisplayFileStatus) Apply(e *editor.Editor) {
	v := e.ActiveView()

	path := v.Buffer().Path
	numLines := v.Buffer().NumLines
	c := v.Cursor()
	pc := (float64(c.LineNum) / float64(numLines)) * 100

	v.SetStatus("\"%s\" %d lines --%d%%--", path, numLines, int(pc))
}
