package main

// main.go is the process entrypoint for the Wolfpack CLI.

import (
	"fmt"
	"os"
)

// main loads configuration, runs the command, and prints any final error.
func main() {
	cfg := loadConfig()
	if err := run(cfg, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
