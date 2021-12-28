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
}
