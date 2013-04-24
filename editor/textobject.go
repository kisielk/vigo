package editor

import (
	"github.com/nsf/termbox-go"
)

type textObjectMode struct {
	editor *editor
	object textObject
}

type textObject struct {
	inner bool
	kind  textObjectKind
}

type textObjectKind int

const (
	textObjectWord textObjectKind = iota
	textObjectWhitespaceWord
	textObjectSentence
	textObjectParagraph
	textObjectSection
	textObjectPercent
	textObjectParens
	textObjectBraces
)

func newTextObjectMode(editor *editor, inner bool) *textObjectMode {
	return &textObjectMode{editor: editor, object: textObject{inner: inner}}
}

func (m *textObjectMode) onKey(ev *termbox.Event) {
	switch ev.Ch {
	case 'w':
		m.object.kind = textObjectWord
	case 'W':
		m.object.kind = textObjectWhitespaceWord
	case 's':
		m.object.kind = textObjectSentence
	case 'p':
		m.object.kind = textObjectParagraph
	case 'S':
		m.object.kind = textObjectSection
	case '%':
		m.object.kind = textObjectPercent
	case 'b':
		m.object.kind = textObjectParens
	case 'B':
		m.object.kind = textObjectBraces
	}
}

func (m *textObjectMode) OnExit() {
	//TODO: Return the text object to the previous mode somehow
}
