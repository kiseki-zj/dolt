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
	pg.flags = leafPageFlag
	ptr := pg
	ptr.count = 3
	data := (*[0x7FFFFFF]uint32)(unsafe.Pointer(&(ptr.ptr)))
	data[0] = 1
	data[1] = 3 * 16
	data[2] = 1
	data[3] = 1
	data[4] = 1
	data[5] = 2*16 + 2
	data[6] = 1
	data[7] = 1
	data[8] = 1
	data[9] = 16 + 4
	data[10] = 1
	data[11] = 1
	kv := (*[0x7FFFFFF]byte)(unsafe.Pointer(&data[12]))
	kv[0] = 'A'
	kv[1] = 'a'
	kv[2] = 'B'
	kv[3] = 'b'
	kv[4] = 'C'
	kv[5] = 'c'

	return ((*[4096]byte)(unsafe.Pointer(pg)))[:]
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
