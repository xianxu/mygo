package main

import (
	/*"fmt"*/
	"unsafe"
	"sync/atomic"
)

func main() {
	a := 1
	b := &a
	c := 2
	atomic.StorePointer(&(unsafe.Pointer)(&b), &c)
}
