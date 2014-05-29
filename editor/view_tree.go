package editor

import (
	"github.com/nsf/tulib"
)

//----------------------------------------------------------------------------
// view_tree
//----------------------------------------------------------------------------

type viewTree struct {
	// At the same time only one of these groups can be valid:
	// 1) 'left', 'right' and 'split'
	// 2) 'top', 'bottom' and 'split'
	// 3) 'leaf'
	parent     *viewTree
	left       *viewTree
	top        *viewTree
	right      *viewTree
	bottom     *viewTree
	leaf       *view
	split      float32
	tulib.Rect // updated with 'resize' call
}

func newViewTreeLeaf(parent *viewTree, v *view) *viewTree {
	return &viewTree{
		parent: parent,
		leaf:   v,
	}
}

func (v *viewTree) splitHorizontally() {
	top := v.leaf
	bottom := newView(top.ctx, top.buf, top.redraw)
	*v = viewTree{
		parent: v.parent,
		top:    newViewTreeLeaf(v, top),
		bottom: newViewTreeLeaf(v, bottom),
		split:  0.5,
	}
}

func (v *viewTree) splitVertically() {
	left := v.leaf
	right := newView(left.ctx, left.buf, left.redraw)
	*v = viewTree{
		parent: v.parent,
		left:   newViewTreeLeaf(v, left),
		right:  newViewTreeLeaf(v, right),
		split:  0.5,
	}
}

func (v *viewTree) draw() {
	if v.leaf != nil {
		v.leaf.draw()
		return
	}

	if v.left != nil {
		v.left.draw()
		v.right.draw()
	} else {
		v.top.draw()
		v.bottom.draw()
	}
}

func (v *viewTree) resize(pos tulib.Rect) {
	v.Rect = pos
	if v.leaf != nil {
		v.leaf.resize(pos.Width, pos.Height)
		return
	}

	if v.left != nil {
		// horizontal split, use 'w'
		w := pos.Width
		if w > 0 {
			// reserve one line for splitter, if we have one line
			w--
		}
		lw := int(float32(w) * v.split)
		rw := w - lw
		v.left.resize(tulib.Rect{pos.X, pos.Y, lw, pos.Height})
		v.right.resize(tulib.Rect{pos.X + lw + 1, pos.Y, rw, pos.Height})
	} else {
		// vertical split, use 'h', no need to reserve one line for
		// splitter, because splitters are part of the buffer's output
		// (their status bars act like a splitter)
		h := pos.Height
		th := int(float32(h) * v.split)
		bh := h - th
		v.top.resize(tulib.Rect{pos.X, pos.Y, pos.Width, th})
		v.bottom.resize(tulib.Rect{pos.X, pos.Y + th, pos.Width, bh})
	}
}

func (v *viewTree) traverse(cb func(*viewTree)) {
	if v.leaf != nil {
		cb(v)
		return
	}

	if v.left != nil {
		v.left.traverse(cb)
		v.right.traverse(cb)
	} else if v.top != nil {
		v.top.traverse(cb)
		v.bottom.traverse(cb)
	}
}

// NearestHSplit returns the viewTree node with the nearest
// horizontally split neighbour view. dir argument controls the search
// direction; -1 searches above the current view, 1 searches below.
// nil is returned if no neighbour is found.
func (v *viewTree) NearestHSplit(dir int) *viewTree {
	w := v.parent
	for w != nil {
		if dir < 0 && w.top != nil && v == w.bottom {
			return w.top.firstLeafNode()
		} else if dir > 0 && w.bottom != nil && v == w.top {
			return w.bottom.firstLeafNode()
		}
		v = w
		w = w.parent
	}
	return nil
}

// NearestVSplit returns the viewTree node with the nearest
// vertically split neighbour view. dir argument controls the search
// direction; -1 searches to the left of current view, 1 searches to the right.
// nil is returned if no neighbour is found.
func (v *viewTree) NearestVSplit(dir int) *viewTree {
	w := v.parent
	for w != nil {
		if dir < 0 && w.left != nil && v == w.right {
			return w.left.firstLeafNode()
		} else if dir > 0 && w.right != nil && v == w.left {
			return w.right.firstLeafNode()
		}
		v = w
		w = w.parent
	}
	return nil
}

func (v *viewTree) oneStep() float32 {
	if v.top != nil {
		return 1.0 / float32(v.Height)
	} else if v.left != nil {
		return 1.0 / float32(v.Width-1)
	}
	return 0.0
}

func (v *viewTree) normalizeSplit() {
	var off int
	if v.top != nil {
		off = int(float32(v.Height) * v.split)
	} else {
		off = int(float32(v.Width-1) * v.split)
	}
	v.split = float32(off) * v.oneStep()
}

func (v *viewTree) stepResize(n int) {
	if v.Width <= 1 || v.Height <= 0 {
		// avoid division by zero, result is really bad
		return
	}

	one := v.oneStep()
	v.normalizeSplit()
	v.split += one*float32(n) + (one * 0.5)
	if v.split > 1.0 {
		v.split = 1.0
	}
	if v.split < 0.0 {
		v.split = 0.0
	}
	v.resize(v.Rect)
}

func (v *viewTree) reparent() {
	if v.left != nil {
		v.left.parent = v
		v.right.parent = v
	} else if v.top != nil {
		v.top.parent = v
		v.bottom.parent = v
	}
}

func (v *viewTree) sibling() *viewTree {
	p := v.parent
	if p == nil {
		return nil
	}
	switch {
	case v == p.left:
		return p.right
	case v == p.right:
		return p.left
	case v == p.top:
		return p.bottom
	case v == p.bottom:
		return p.top
	}
	panic("unreachable")
}

func (v *viewTree) firstLeafNode() *viewTree {
	if v.left != nil {
		return v.left.firstLeafNode()
	} else if v.top != nil {
		return v.top.firstLeafNode()
	} else if v.leaf != nil {
		return v
	}
	panic("unreachable")
}
