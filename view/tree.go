package view

import (
	"github.com/nsf/tulib"
)

type Tree struct {
	// At the same time only one of these groups can be valid:
	// 1) 'left', 'right' and 'split'
	// 2) 'top', 'bottom' and 'split'
	// 3) 'leaf'
	parent     *Tree
	left       *Tree
	top        *Tree
	right      *Tree
	bottom     *Tree
	leaf       *View
	split      float32
	tulib.Rect // updated with 'resize' call
}

func (t *Tree) Leaf() *View {
	return t.leaf
}

func NewTree(v *View) *Tree {
	return &Tree{leaf: v}
}

func NewTreeLeaf(parent *Tree, v *View) *Tree {
	return &Tree{
		parent: parent,
		leaf:   v,
	}
}

func (t *Tree) NewLeaf(v *View) *Tree {
	return &Tree{
		parent: t,
		leaf:   v,
	}
}

func (t *Tree) Parent() *Tree {
	return t.parent
}

func (t *Tree) Left() *Tree {
	return t.left
}

func (t *Tree) Right() *Tree {
	return t.right
}

func (t *Tree) Top() *Tree {
	return t.top
}

func (t *Tree) Bottom() *Tree {
	return t.bottom
}

func (v *Tree) SplitHorizontally() {
	top := v.leaf
	bottom := NewView(top.ctx, top.buf, top.redraw)
	*v = Tree{
		parent: v.parent,
		split:  0.5,
	}
	v.top = v.NewLeaf(top)
	v.bottom = v.NewLeaf(bottom)
}

func (t *Tree) SetParent(parent *Tree) {
	t.parent = parent
}

func (v *Tree) SplitVertically() {
	left := v.leaf
	right := NewView(left.ctx, left.buf, left.redraw)
	*v = Tree{
		parent: v.parent,
		split:  0.5,
	}
	v.left = v.NewLeaf(left)
	v.right = v.NewLeaf(right)
}

func (v *Tree) Draw() {
	if v.leaf != nil {
		v.leaf.draw()
		return
	}

	if v.left != nil {
		v.left.Draw()
		v.right.Draw()
	} else {
		v.top.Draw()
		v.bottom.Draw()
	}
}

func (v *Tree) Resize(pos tulib.Rect) {
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
		v.left.Resize(tulib.Rect{pos.X, pos.Y, lw, pos.Height})
		v.right.Resize(tulib.Rect{pos.X + lw + 1, pos.Y, rw, pos.Height})
	} else {
		// vertical split, use 'h', no need to reserve one line for
		// splitter, because splitters are part of the buffer's output
		// (their status bars act like a splitter)
		h := pos.Height
		th := int(float32(h) * v.split)
		bh := h - th
		v.top.Resize(tulib.Rect{pos.X, pos.Y, pos.Width, th})
		v.bottom.Resize(tulib.Rect{pos.X, pos.Y + th, pos.Width, bh})
	}
}

func (v *Tree) Walk(cb func(*Tree)) {
	if v.leaf != nil {
		cb(v)
		return
	}

	if v.left != nil {
		v.left.Walk(cb)
		v.right.Walk(cb)
	} else if v.top != nil {
		v.top.Walk(cb)
		v.bottom.Walk(cb)
	}
}

// NearestHSplit returns the Tree node with the nearest
// horizontally split neighbour view. dir argument controls the search
// direction; -1 searches above the current view, 1 searches below.
// nil is returned if no neighbour is found.
func (v *Tree) NearestHSplit(dir int) *Tree {
	w := v.parent
	for w != nil {
		if dir < 0 && w.top != nil && v == w.bottom {
			return w.top.FirstLeafNode()
		} else if dir > 0 && w.bottom != nil && v == w.top {
			return w.bottom.FirstLeafNode()
		}
		v = w
		w = w.parent
	}
	return nil
}

// NearestVSplit returns the Tree node with the nearest
// vertically split neighbour view. dir argument controls the search
// direction; -1 searches to the left of current view, 1 searches to the right.
// nil is returned if no neighbour is found.
func (v *Tree) NearestVSplit(dir int) *Tree {
	w := v.parent
	for w != nil {
		if dir < 0 && w.left != nil && v == w.right {
			return w.left.FirstLeafNode()
		} else if dir > 0 && w.right != nil && v == w.left {
			return w.right.FirstLeafNode()
		}
		v = w
		w = w.parent
	}
	return nil
}

func (v *Tree) oneStep() float32 {
	if v.top != nil {
		return 1.0 / float32(v.Height)
	} else if v.left != nil {
		return 1.0 / float32(v.Width-1)
	}
	return 0.0
}

func (v *Tree) normalizeSplit() {
	var off int
	if v.top != nil {
		off = int(float32(v.Height) * v.split)
	} else {
		off = int(float32(v.Width-1) * v.split)
	}
	v.split = float32(off) * v.oneStep()
}

func (v *Tree) stepResize(n int) {
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
	v.Resize(v.Rect)
}

func (v *Tree) Reparent(parent *Tree) {
	v.parent = parent
	if v.left != nil {
		v.left.parent = v
		v.right.parent = v
	} else if v.top != nil {
		v.top.parent = v
		v.bottom.parent = v
	}
}

func (v *Tree) Sibling() *Tree {
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

func (v *Tree) FirstLeafNode() *Tree {
	if v.left != nil {
		return v.left.FirstLeafNode()
	} else if v.top != nil {
		return v.top.FirstLeafNode()
	} else if v.leaf != nil {
		return v
	}
	panic("unreachable")
}
