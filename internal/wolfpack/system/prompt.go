package system

// prompt.go handles interactive input, yes/no prompts, and secret entry.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// NewInputReader creates a stdin reader for one prompt sequence.
func NewInputReader() *bufio.Reader {
	return bufio.NewReader(os.Stdin)
}

// PromptLine reads one line, showing a prompt only for interactive terminals.
func PromptLine(prompt string) string {
	if StdinIsTTY() {
		fmt.Fprint(os.Stderr, prompt)
	}
	reader := NewInputReader()
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

// PromptSecret reads a sensitive value with terminal echo disabled when possible.
func PromptSecret(prompt string) string {
	if !StdinIsTTY() {
		return PromptLine("")
	}
	fmt.Fprint(os.Stderr, prompt)
	if HaveCmd("stty") {
		_ = RunCommandWithIO("stty", []string{"-echo"}, os.Stdin, io.Discard, io.Discard)
		defer func() {
			_ = RunCommandWithIO("stty", []string{"echo"}, os.Stdin, io.Discard, io.Discard)
			fmt.Fprintln(os.Stderr)
		}()
	}
	reader := NewInputReader()
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

// PromptYesNo asks a yes/no question and uses the default in non-TTY mode.
func PromptYesNo(prompt string, defaultYes bool) bool {
	if !StdinIsTTY() {
		return defaultYes
	}
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	reader := NewInputReader()
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
		Warn("answer yes or no")
	}
}
