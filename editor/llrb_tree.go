package editor

import (
	"bytes"
)

// LLRB tree with single key values as byte slices.
// I use 2-3 tree algorithms for it. Only insertion is implemented (no delete).
type llrbTree struct {
	root      *llrbNode
	count     int
	freeNodes *llrbNode
}

func (t *llrbTree) freeNode(n *llrbNode) {
	*n = llrbNode{left: t.freeNodes}
	t.freeNodes = n
}

func (t *llrbTree) allocNode(value []byte) *llrbNode {
	if t.freeNodes == nil {
		return &llrbNode{value: value}
	}

	n := t.freeNodes
	t.freeNodes = n.left
	*n = llrbNode{value: value}
	return n
}

func (t *llrbTree) clear() {
	t.clearRecursive(t.root)
	t.root = nil
	t.count = 0
}

func (t *llrbTree) clearRecursive(n *llrbNode) {
	if n == nil {
		return
	}
	t.clearRecursive(n.left)
	t.clearRecursive(n.right)
	t.freeNode(n)
}

func (t *llrbTree) walk(cb func(value []byte)) {
	t.root.walk(cb)
}

func (t *llrbTree) insertMaybe(value []byte) bool {
	var ok bool
	t.root, ok = t.root.insertMaybe(value)
	if ok {
		t.count++
	}
	return ok
}

func (t *llrbTree) insertMaybeRecursive(n *llrbNode, value []byte) (*llrbNode, bool) {
	if n == nil {
		return t.allocNode(value), true
	}

	var inserted bool
	switch cmp := bytes.Compare(value, n.value); {
	case cmp < 0:
		n.left, inserted = t.insertMaybeRecursive(n.left, value)
	case cmp > 0:
		n.right, inserted = t.insertMaybeRecursive(n.right, value)
	default:
		// don't insert anything
	}

	if n.right.isRed() && !n.left.isRed() {
		n = n.rotateLeft()
	}
	if n.left.isRed() && n.left.left.isRed() {
		n = n.rotateRight()
	}
	if n.left.isRed() && n.right.isRed() {
		n.flipColors()
	}

	return n, inserted
}

func (t *llrbTree) contains(value []byte) bool {
	return t.root.contains(value)
}

const (
	llrbRed   = false
	llrbBlack = true
)

type llrbNode struct {
	value []byte
	left  *llrbNode
	right *llrbNode
	color bool
}

func (n *llrbNode) walk(cb func(value []byte)) {
	if n == nil {
		return
	}
	n.left.walk(cb)
	cb(n.value)
	n.right.walk(cb)
}

func (n *llrbNode) rotateLeft() *llrbNode {
	x := n.right
	n.right = x.left
	x.left = n
	x.color = n.color
	n.color = llrbRed
	return x
}

func (n *llrbNode) rotateRight() *llrbNode {
	x := n.left
	n.left = x.right
	x.right = n
	x.color = n.color
	n.color = llrbRed
	return x
}

func (n *llrbNode) flipColors() {
	n.color = !n.color
	n.left.color = !n.left.color
	n.right.color = !n.right.color
}

func (n *llrbNode) isRed() bool {
	return n != nil && !n.color
}

func (n *llrbNode) insertMaybe(value []byte) (*llrbNode, bool) {
	if n == nil {
		return &llrbNode{value: value}, true
	}

	var inserted bool
	switch cmp := bytes.Compare(value, n.value); {
	case cmp < 0:
		n.left, inserted = n.left.insertMaybe(value)
	case cmp > 0:
		n.right, inserted = n.right.insertMaybe(value)
	default:
		// don't insert anything
	}

	if n.right.isRed() && !n.left.isRed() {
		n = n.rotateLeft()
	}
	if n.left.isRed() && n.left.left.isRed() {
		n = n.rotateRight()
	}
	if n.left.isRed() && n.right.isRed() {
		n.flipColors()
	}

	return n, inserted
}

func (n *llrbNode) contains(value []byte) bool {
	for n != nil {
		switch cmp := bytes.Compare(value, n.value); {
		case cmp < 0:
			n = n.left
		case cmp > 0:
			n = n.right
		default:
			return true
		}
	}
	return false
}
