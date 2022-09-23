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
	// скобки в паттерне вызывают проблемы - тест не может сматчить. Не могу найти информацию как экранировать
	os.Exit(0) // want "call os.Exit in main function of package main"
}
