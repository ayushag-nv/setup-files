package main

// main.go is the process entrypoint for the Wolfpack CLI.

import (
	"fmt"
	"os"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack"
)

// main delegates to the internal Wolfpack application package.
func main() {
	if err := wolfpack.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
