package dolt

import (
	"fmt"
	"testing"
	"unsafe"
)

func TestReadPage(t *testing.T) {
	db, _ := Open("./testdb", 0666)
	page := (*page)(unsafe.Pointer(&db.data[0]))
	nd := new(node)
	nd.read(page)
	fmt.Println(len(nd.inodes))
	fmt.Println(nd.inodes)
	db.close()
}

func iTestPut(t *testing.T) {
	db, _ := Open("./testdb", 0666)
	page := (*page)(unsafe.Pointer(&db.data[0]))
	nd := new(node)
	nd.read(page)
	t.Log(len(nd.inodes))
	t.Log(nd.inodes)
	oldkey := []byte{'D'}
	newkey := []byte{'D'}
	value := []byte{'d', 'd', 'd', 'd'}

	nd.put(oldkey, newkey, value, 0, 1)
	t.Log(len(nd.inodes))
	t.Log(nd.inodes)
	delkey := []byte{'B'}
	nd.del(delkey)
	t.Log(len(nd.inodes))
	t.Log(nd.inodes)
	db.close()
}

func iTestWrite(t *testing.T) {
	db, _ := Open("./testdb", 0666)
	//再写一个空的page扩展db大小
	var pg = make([]byte, 4096)
	db.file.WriteAt(pg, 4096)
	pagee := (*page)(unsafe.Pointer(&db.data[0]))
	nd := new(node)
	nd.read(pagee)
	oldKey := []byte{'D'}
	newKey := []byte{'D'}
	value := []byte{'d', 'd', 'd', 'd'}
	nd.put(oldKey, newKey, value, 0, 0)
	t.Log(len(nd.inodes))
	t.Log(nd.inodes)
	nd.isLeaf = true
	t.Log(pagee.flags)
	//node1=page1，修改node1
	page2 := (*page)(unsafe.Pointer(&db.data[4096]))
	//把node1的内容写给page2
	nd.write(page2)
	nd2 := new(node)
	//查看page2
	nd2.read(page2)
	t.Log(len(nd2.inodes))
	t.Log(nd2.inodes)
	db.close()
}

func TestContent(t *testing.T) {
	db, _ := Open("./testdb", 0666)
	pg1 := (*page)(unsafe.Pointer(&db.data[0]))
	pg2 := (*page)(unsafe.Pointer(&db.data[4096]))
	t.Log(pg1.count)
	for i := 0; i < int(pg1.count); i++ {
		elem := pg1.leafPageElement(uint16(i))
		t.Log(elem.key())
		t.Log(elem.value())
	}
	t.Log(pg2.count)
	for i := 0; i < int(pg2.count); i++ {
		elem := pg2.leafPageElement(uint16(i))
		t.Log(elem.key())
		t.Log(elem.value())
	}
	db.close()
}
