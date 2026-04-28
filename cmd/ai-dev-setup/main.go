package main

import (
	"fmt"
	"os"
)

func main() {
	cfg := loadConfig()
	if err := run(cfg, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
