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

/*func (n *node) childAt(index int) *node {
	if n.isLeaf {
		panic(fmt.Sprintf("invalid childAt(%d) on a leaf node", index))
	}
	return n.bucket.node(n.inodes[index].pgid, n)
}

func (n *node) childIndex(child *node) int {
	index := sort.Search(len(n.inodes), func(i int) bool { return bytes.Compare(n.inodes[i], key, child.key) != -1 })
	return index
}*/

func (n *node) numChildren() int {
	return len(n.inodes)
}

// nextSibling returns the next node with the same parent.
/*func (n *node) nextSibling() *node {
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
}*/

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
