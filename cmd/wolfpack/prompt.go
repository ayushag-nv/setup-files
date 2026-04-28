package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func newInputReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}

func promptLine(prompt string) string {
	if stdinIsTTY() {
		fmt.Fprint(os.Stderr, prompt)
	}
	reader := newInputReader()
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

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
