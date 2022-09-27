package main

import (
	"fmt"
	"os"
)

func testfunc() {
	fmt.Println("123")
	os.Exit(0)
}

func main() {
	testfunc()
}
