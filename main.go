// main.go
package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("tui-ssm %s\n", version)
		os.Exit(0)
	}
	fmt.Println("tui-ssm: starting...")
}
