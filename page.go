package dolt

import (
	"fmt"
	"unsafe"
)

type pgid uint64

type page struct {
	id       pgid
	flags    uint16
	count    uint16
	overflow uint32
	ptr      *byte //不建议使用uintptr
}

type pages page
type pgids []pgid

type branchPageElement struct {
	pos   uint32
	ksize uint32
	pgid  pgid
}

type leafPageElement struct {
	flags uint32
	pos   uint32
	ksize uint32
	vsize uint32
}

const pageHeaderSize = int(unsafe.Offsetof((*page)(nil).ptr))

const minKeysPerPage = 2

const branchPageElementSize = int(unsafe.Sizeof(branchPageElement{}))
const leafPageElementSize = int(unsafe.Sizeof(leafPageElement{}))

const (
	branchPageFlag   = 0x01
	leafPageFlag     = 0x02
	metaPageFlag     = 0x04
	freelistPageFlag = 0x10
)

const (
	bucketLeafFlag = 0x01
)

func (p *page) typ() string {
	if (p.flags & branchPageFlag) != 0 {
		return "branch"
	} else if (p.flags & leafPageFlag) != 0 {
		return "leaf"
	} else if (p.flags & metaPageFlag) != 0 {
		return "meta"
	} else if (p.flags & freelistPageFlag) != 0 {
		return "freelist"
	}
	return fmt.Sprintf("unknown<%02x>", p.flags)
}

/*
func (p *page) meta() *meta {
	return (*meta)(un)
}
*/

func (p *page) leafPageElement(index uint16) *leafPageElement {
	offset := index * uint16(leafPageElementSize)
	res := (*leafPageElement)(unsafe.Pointer(((uintptr)(unsafe.Pointer(&p.ptr)) + (uintptr)(offset)))) //uintptr可以做指针算术运算
	return res
}

func (p *page) leafPageElements() []leafPageElement {
	if p.count == 0 {
		return nil
	}
	return ((*[0x7FFFFFF]leafPageElement)(unsafe.Pointer(&p.ptr)))[:] //转array，生成slice，slice的len和cap为0x7ffffff
}

func (p *page) branchPageElement(index uint16) *branchPageElement {
	offset := index * uint16(branchPageElementSize)
	res := (*branchPageElement)(unsafe.Pointer(((uintptr)(unsafe.Pointer(&p.ptr)) + (uintptr)(offset))))
	return res
}

func (p *page) branchPageElements() []branchPageElement {
	if p.count == 0 {
		return nil
	}
	return ((*[0x7FFFFFF]branchPageElement)(unsafe.Pointer(&p.ptr)))[:]
}

func (s pgids) Len() int           { return len(s) }
func (s pgids) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s pgids) Less(i, j int) bool { return s[i] < s[j] }

func (n *branchPageElement) key() []byte {
	ptr := (*[0x7FFFFFF]byte)(unsafe.Pointer(((uintptr)(unsafe.Pointer(n)) + (uintptr)(n.pos))))
	//不能写成*[]byte,如果这样写，ptr指向一个SliceHeader
	return ptr[:n.ksize]
}

func (n *leafPageElement) key() []byte {
	ptr := (*[0x7FFFFFF]byte)(unsafe.Pointer(((uintptr)(unsafe.Pointer(n)) + (uintptr)(n.pos))))
	return ptr[:n.ksize]
}

func (n *leafPageElement) value() []byte {
	ptr := (*[0x7FFFFFF]byte)(unsafe.Pointer(((uintptr)(unsafe.Pointer(n)) + (uintptr)(n.pos))))
	return ptr[n.ksize : n.ksize+n.vsize]
}
