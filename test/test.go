package main

import (
	"fmt"
)

type tt struct {
	a uint64
	b uint64
}

func main() {
	var a int = 5
	var p uintptr = (uintptr)(&a)
	fmt.Println(p)
	fmt.Println(*p)
}
