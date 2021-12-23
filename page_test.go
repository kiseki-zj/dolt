package dolt

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestA(t *testing.T) {
	bytes := new([4096]byte)
	var p = (*page)(unsafe.Pointer(bytes))
	p.count = 10
	e := p.leafPageElement(0)
	e.flags = 123
	fmt.Printf("%d %d %p %p %p\n", pageHeaderSize, p.ptr, p, &p.ptr, e)
	es := p.leafPageElements()
	fmt.Printf("%p %d %d\n", es, len(es), cap(es))
	fmt.Printf("%d\n", es[0].flags)
	fmt.Println(es[12345667])
}
