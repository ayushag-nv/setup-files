package system

// Package system contains OS, shell, prompt, and filesystem helpers.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

// ShellQuote quotes a value for safe single-quoted shell usage.
func ShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// Warn prints a warning message to stderr.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// StdinIsTTY reports whether stdin is an interactive terminal.
func StdinIsTTY() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// AssertSupportedOS rejects platforms outside the macOS/Linux support scope.
func AssertSupportedOS() error {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		return nil
	}
	return fmt.Errorf("unsupported OS: %s. This CLI supports macOS and Linux", runtime.GOOS)
}

// DownloadFile downloads a URL to a local path.
func DownloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
