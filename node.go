package dolt

type node struct {
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
