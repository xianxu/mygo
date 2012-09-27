package main

import (
	"fmt"
)

func main() {
	a := map[string]string { "a": "a" }
	b := map[string]string { "b": "b" }

	fmt.Printf("%v", a + b)
}
