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
	ptr      *byte
}

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
	res := (*leafPageElement)(unsafe.Pointer(((uintptr)(unsafe.Pointer(&p.ptr)) + (uintptr)(offset))))
	return res
}

func (p *page) leafPageElements() []leafPageElement {
	if p.count == 0 {
		return nil
	}
	return ((*[0x7FFFFFF]leafPageElement)(unsafe.Pointer(&p.ptr)))[:]
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
