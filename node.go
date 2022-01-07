package dolt

import (
	"bytes"
	"fmt"
	"sort"
	"unsafe"
)

type node struct {
	//bucket     *Bucket
	isLeaf     bool
	unbalanced bool
	spilled    bool
	key        []byte
	pgid       pgid
	parent     *node
	children   nodes
	inodes     inodes
}

type nodes []*node

func (s nodes) Len() int      { return len(s) }
func (s nodes) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s nodes) Less(i, j int) bool {
	return bytes.Compare(s[i].inodes[0].key, s[j].inodes[0].key) == -1
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

func (n *node) numChildren() int {
	return len(n.inodes)
}

// nextSibling returns the next node with the same parent.
func (n *node) nextSibling() *node {
	if n.parent == nil {
		return nil
	}
	index := n.parent.childIndex(n)
	if index >= n.parent.numChildren()-1 {
		return nil
	}
	return n.parent.childAt(index + 1)
}

// prevSibling returns the previous node with the same parent.
func (n *node) prevSibling() *node {
	if n.parent == nil {
		return nil
	}
	index := n.parent.childIndex(n)
	if index == 0 {
		return nil
	}
	return n.parent.childAt(index - 1)
}

//oldkey=newkey，oldkey存在，是修改value
//oldkey不存在，是插入
//oldkey!=newkey，oldkey存在，是把(oldkey,oldvalue)修改为(newkey,value)
func (n *node) put(oldKey, newKey, value []byte, pgid pgid, flags uint32) {
	/*
		if pgid >= n.bucket.tx.meta.pgid {
			panic(fmt.Sprintf("pgid (%d) above high water mark (%d)", pgid, n.bucket.tx.meta.pgid))
		} else if len(oldKey) <= 0 {
			panic("put: zero-length old key")
		} else if len(newKey) <= 0 {
			panic("put: zero-length new key")
		}
	*/
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i].key, oldKey) != -1 })
	exact := (len(n.inodes) > 0 && index < len(n.inodes) && bytes.Equal(n.inodes[index].key, oldKey))
	//exact:oldkey是否存在
	if !exact {
		n.inodes = append(n.inodes, inode{}) //inodes的len，cap初始为p.count，append会找寻新的连续空间，但并不影响结果，
		//因为扩充的是inodes这个inode切片，inode切片的pointer是指向page的指针，这样扩充不会影响到page，
		copy(n.inodes[index+1:], n.inodes[index:])
	}
	inode := &n.inodes[index]
	inode.flags = flags
	inode.key = newKey
	inode.value = value
	inode.pgid = pgid
	//这里是COW，原本inode的key和value都指向page mmap的地址，如果
	//有新kv或修改v，会指向新创建的newKey和value
	_assert(len(inode.key) > 0, "put: zero-length inode key")
}

func (n *node) del(key []byte) {
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i].key, key) != -1 })
	if index >= len(n.inodes) || !bytes.Equal(n.inodes[index].key, key) {
		return
	}
	n.inodes = append(n.inodes[:index], n.inodes[index+1:]...)
	n.unbalanced = true
}

func (n *node) read(p *page) {
	n.pgid = p.id
	n.isLeaf = ((p.flags & leafPageFlag) != 0)
	n.inodes = make(inodes, int(p.count))
	for i := 0; i < int(p.count); i++ {
		inode := &n.inodes[i]
		if n.isLeaf {
			elem := p.leafPageElement(uint16(i))
			inode.key = elem.key()
			inode.value = elem.value()
		} else {
			elem := p.branchPageElement(uint16(i))
			inode.key = elem.key()
			inode.pgid = elem.pgid
		}
		_assert(len(inode.key) > 0, "read: zero-length inode key")
	}
	if len(n.inodes) > 0 {
		n.key = n.inodes[0].key
		_assert(len(n.key) > 0, "read: zero-length node key")
	} else {
		n.key = nil
	}
}

func (n *node) write(p *page) {
	if n.isLeaf {
		p.flags |= leafPageFlag
	} else {
		p.flags |= branchPageFlag
	}

	if len(n.inodes) >= 0xFFFF {
		panic(fmt.Sprintf("inode overflow: %d (pgid=%d)", len(n.inodes), p.id))
	}
	p.count = (uint16)(len(n.inodes))
	if p.count == 0 {
		return
	}
	b := (*[0x7FFFFFFF]byte)(unsafe.Pointer(&p.ptr))[n.pageElementSize()*len(n.inodes):]
	for i, item := range n.inodes {
		_assert(len(item.key) > 0, "write: zero-length inode key")
		if n.isLeaf {
			elem := p.leafPageElement(uint16(i))
			elem.pos = (uint32)(uintptr(unsafe.Pointer(&b[0])) - uintptr(unsafe.Pointer(elem)))
			elem.flags = item.flags
			elem.ksize = uint32(len(item.key))
			elem.vsize = uint32(len(item.value))
		} else {
			elem := p.branchPageElement(uint16(i))
			elem.pos = (uint32)(uintptr(unsafe.Pointer(&b[0])) - uintptr(unsafe.Pointer(elem)))
			elem.ksize = uint32(len(item.key))
			elem.pgid = item.pgid
			_assert(elem.pgid != p.id, "write: cicular dependency occurred")
		}
		klen, vlen := len(item.key), len(item.value)
		if len(b) < klen+vlen {
			b = (*[0x7FFFFFFF]byte)(unsafe.Pointer(&b[0]))[:]
		}

		copy(b[0:], item.key)
		b = b[klen:]
		copy(b[0:], item.value)
		b = b[vlen:]
	}
}

func (n *node) split(pageSize int) []*node {
	var nodes []*node
	node := n
	for {
		a, b := node.splitTwo(pageSize)
		nodes = append(nodes, a)
		if b == nil {
			break
		}
		node = b
	}
	return nodes
}

func (n *node) splitTwo(pageSize int) (*node, *node) {
	if len(n.inodes) <= (minKeysPerPage*2) || n.sizeLessThan(pageSize) {
		return n, nil
	}
	//var fillPercent = n.bucket.FillPercent
	var fillPercent = 0.5
	//if fillPercent < minFillPercent
	threshold := int(float64(pageSize) * fillPercent)
	splitIndex, _ := n.splitIndex(threshold)
	if n.parent == nil {
		n.parent = &node{children: []*node{n}}
	}

	next := &node{isLeaf: n.isLeaf, parent: n.parent}
	n.parent.children = append(n.parent.children, next)
	next.inodes = n.inodes[splitIndex:]
	n.inodes = n.inodes[:splitIndex]
	//n.bucket.tx.stats.Split++

	return n, next
}

func (n *node) splitIndex(threshold int) (index, sz int) {
	sz = pageHeaderSize
	for i := 0; i < len(n.inodes)-minKeysPerPage; i++ {
		index = i
		inode := n.inodes[i]
		elsize := n.pageElementSize() + len(inode.key) + len(inode.value)

		if i >= minKeysPerPage && sz+elsize > threshold {
			break
		}
		sz += elsize
	}

	return
}

func (n *node) spill() error {
	//var tx = n.bucket.tx
	if n.spilled {
		return nil
	}

	sort.Sort(n.children)
	for i := 0; i < len(n.children); i++ {
		if err := n.children[i].spill(); err != nil {
			return err
		}
	}
	n.children = nil

	var nodes = n.split(4096)
	for _, node := range nodes {
		if node.pgid > 0 {
			//tx.db.freelist.free(tx.meta.txid, tx.page(node.pgid))
			node.pgid = 0
		}
		/*p, err := tx.allocate((node.size() / tx.db.pageSize) + 1)
		if err != nil {
			return nil
		}*/
		if p.id >= tx.meta.pgid {
			panic(fmt.Sprintf("pgid (%d) above high water mark (%d)", p.id, tx.meta.pgid))
		}
		node.pgid = p.id
		node.write(p)
		node.spilled = true

		if node.parent != nil {
			var key = node.key
			if key == nil {
				key = node.inodes[0].key
			}

			node.parent.put(key, node.inodes[0].key, nil, node.pgid, 0)
			node.key = node.inodes[0].key
			_assert(len(node.key) > 0, "spill: zero-length node key")
		}

		//tx.stats.Spill++
	}
	if n.parent != nil && n.parent.pgid == 0 {
		n.children = nil
		return n.parent.spill()
	}
	return nil
}

func (n *node) rebalance() {
	if !n.unbalanced {
		return
	}
	n.unbalanced = false

	//n.bucket.tx.stats.Rebalance++

	//var threshold = n.bucket.tx.db.pageSize / 4
	var threshold = 4096 / 4
	if n.size() > threshold && len(n.inodes) > n.minKeys() {
		return
	}
}
