package main

import (
	"tfe"
	"fmt"
)

func main() {
	reader := &tfe.CachedReader { []byte("hello\n"), 0, false }
	buf := make([]byte, 100)
	reader.Read(buf)
	fmt.Println(string(buf))
	buf = make([]byte, 100)
	reader.Reset()
	reader.Read(buf)
	fmt.Println(string(buf))
}
