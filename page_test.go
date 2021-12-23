package dolt

import (
	"fmt"
	"testing"
	"unsafe"
)

func makepage() []byte {
	var P [4096]byte
	pg := (*page)(unsafe.Pointer(&P))
	pg.id = 1

	ptr := pg
	ptr.count = 1
	e := ptr.branchPageElement(0)
	e.pos = 123
	e.ksize = 345
	e.pgid = 11
	return ((*[4096]byte)(unsafe.Pointer(ptr)))[:]
}
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
}

func TestOpen(t *testing.T) {
	db, _ := Open("./testdb", 0666)
	pg := makepage()
	db.file.Write(pg)
	ptr := (*page)(unsafe.Pointer(&db.data[0]))
	res := ptr.branchPageElement(0) //test mmap & branchPageElement
	t.Log(res)
	t.Logf("%d %d %d\n", res.pos, res.ksize, res.pgid)
	ress := ptr.branchPageElements()
	t.Log(ress[0])
	t.Log(ress[1])
	db.close()
}
