package main

import (
	"fmt"
	"unsafe"
)

type tt struct {
	a uint64
	b uint64
}

func main() {
	var arr [10]int
	var parr *[10]int = &arr
	fmt.Printf("%p %p\n", arr, parr)
	fmt.Println(unsafe.Sizeof(int(1)))
}
