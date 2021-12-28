package main

import (
	"fmt"
	"os"
	"unsafe"
)

type tt struct {
	a uint64
	b uint64
}
type bs []byte

func main() {
	var nd [10]int
	nd[0] = 'a'
	nd[1] = 'b'
	nd[2] = 'c'
	nd[3] = 'd'
	nd[4] = 'e'
	nd[5] = 'f'
	b := (*[]byte)(unsafe.Pointer(&nd[0]))
	fmt.Printf("%p %p\n", b, &nd[0])
	fmt.Println(len(*b))
	file, _ := os.OpenFile("./testdb", os.O_RDWR, 0666)
	bs := make([]byte, 4096)
	n, _ := file.Read(bs)
	fmt.Println(n)
	fmt.Println(bs)
}
