package dolt

import (
	"bytes"
	"fmt"
	"sort"
)

type node struct {
	bucket     *Bucket
	isLeaf     bool
	unbalanced bool
	spilled    bool
	key        []byte
	pgid       pgid
	parent     *node
	children   []node
	inodes     inodes
}

type inode struct {
	flags uint32
	pgid  pgid
	key   []byte
	value []byte
}

type inodes []inode

func (n *node) root() *node {
	if n.parent == nil {
		return n
	}
	return n.parent.root()
}

func (n *node) minKeys() int {
	if n.isLeaf {
		return 1
	}
	return 2
}

func (n *node) size() int {
	sz, elsz := pageHeaderSize, n.pageElementSize()
	for i := 0; i < len(n.inodes); i++ {
		item := &n.inodes[i]
		sz += elsz + len(item.key) + len(item.value)
	}
	return sz
}

func (n *node) sizeLessThan(v int) bool {
	sz, elsz := pageHeaderSize, n.pageElementSize()
	for i := 0; i < len(n.inodes); i++ {
		item := &n.inodes[i]
		sz += elsz + len(item.key) + len(item.value)
		if sz >= v {
			return false
		}
	}
	return true
}

func (n *node) pageElementSize() int {
	if n.isLeaf {
		return leafPageElementSize
	}
	return branchPageElementSize
}

func (n *node) childAt(index int) *node {
	if n.isLeaf {
		panic(fmt.Sprintf("invalid childAt(%d) on a leaf node", index))
	}
	return n.bucket.node(n.inodes[index].pgid, n)
}

func (n *node) childIndex(child *node) int {
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i], key, child.key) != -1 })
	return index
}
