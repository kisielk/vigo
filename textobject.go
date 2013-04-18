package main

import (
	"github.com/nsf/termbox-go"
)

type TextObjectMode struct {
	stub_overlay_mode
	editor *editor
	object TextObject
}

type TextObject struct {
	Inner bool
	Kind  TextObjectKind
}

type TextObjectKind int

const (
	TextObjectWord TextObjectKind = iota
	TextObjectWhitespaceWord
	TextObjectSentence
	TextObjectParagraph
	TextObjectSection
	TextObjectPercent
	TextObjectParens
	TextObjectBraces
)

func NewTextObjectMode(editor *editor, inner bool) *TextObjectMode {
	return &TextObjectMode{editor: editor, object: TextObject{Inner: inner}}
}

func (m *TextObjectMode) OnKey(ev *termbox.Event) {
	switch ev.Ch {
	case 'w':
		m.object.Kind = TextObjectWord
	case 'W':
		m.object.Kind = TextObjectWhitespaceWord
	case 's':
		m.object.Kind = TextObjectSentence
	case 'p':
		m.object.Kind = TextObjectParagraph
	case 'S':
		m.object.Kind = TextObjectSection
	case '%':
		m.object.Kind = TextObjectPercent
	case 'b':
		m.object.Kind = TextObjectParens
	case 'B':
		m.object.Kind = TextObjectBraces
	}
}

func (m *TextObjectMode) OnExit() {
	//TODO: Return the text object to the previous mode somehow
}
