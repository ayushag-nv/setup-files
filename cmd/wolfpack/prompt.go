package main

// prompt.go handles interactive input, yes/no prompts, and secret entry.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// newInputReader creates a stdin reader for one prompt sequence.
func newInputReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}

// promptLine reads one line, showing a prompt only for interactive terminals.
func promptLine(prompt string) string {
	if stdinIsTTY() {
		fmt.Fprint(os.Stderr, prompt)
	}
	reader := newInputReader()
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

// promptSecret reads a sensitive value with terminal echo disabled when possible.
func promptSecret(prompt string) string {
	if !stdinIsTTY() {
		return promptLine("")
	}
	fmt.Fprint(os.Stderr, prompt)
	if haveCmd("stty") {
		_ = runCommandWithIO("stty", []string{"-echo"}, os.Stdin, io.Discard, io.Discard)
		defer func() {
			_ = runCommandWithIO("stty", []string{"echo"}, os.Stdin, io.Discard, io.Discard)
			fmt.Fprintln(os.Stderr)
		}()
	}
	reader := newInputReader()
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

// promptYesNo asks a yes/no question and uses the default in non-TTY mode.
func promptYesNo(prompt string, defaultYes bool) bool {
	if !stdinIsTTY() {
		return defaultYes
	}
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	reader := newInputReader()
	for {
		fmt.Fprintf(os.Stderr, "%s %s ", prompt, suffix)
		answer, _ := reader.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "" {
			return defaultYes
		}
		if answer == "y" || answer == "yes" {
			return true
		}
		if answer == "n" || answer == "no" {
			return false
		}
		warn("answer yes or no")
	}
}
