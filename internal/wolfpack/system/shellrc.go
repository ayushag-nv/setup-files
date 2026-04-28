package system

// shellrc.go owns all managed edits to shell startup files.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
)

const (
	beginMarker  = "# >>> wolfpack managed env >>>"
	endMarker    = "# <<< wolfpack managed env <<<"
	wrapperBegin = "# >>> wolfpack shell wrapper >>>"
	wrapperEnd   = "# <<< wolfpack shell wrapper <<<"
)

// ensureManagedBlock creates the rc-file block used for Wolfpack env vars.
func ensureManagedBlock(cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.RCFile), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.RCFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if strings.Contains(string(content), beginMarker) {
		return nil
	}
	f, err := os.OpenFile(cfg.RCFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n%s\n", beginMarker, endMarker)
	return err
}

// UpsertEnvVar inserts or replaces one exported variable in the managed block.
func UpsertEnvVar(cfg config.Config, key, value string) error {
	if err := ensureManagedBlock(cfg); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.RCFile)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("export %s=%s", key, ShellQuote(value))
	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	var output []string
	inBlock := false
	seen := false
	for _, current := range lines {
		switch current {
		case beginMarker:
			inBlock = true
			output = append(output, current)
			continue
		case endMarker:
			if !seen {
				output = append(output, line)
			}
			inBlock = false
			output = append(output, current)
			continue
		}
		if inBlock && strings.HasPrefix(current, "export "+key+"=") {
			if !seen {
				output = append(output, line)
				seen = true
			}
			continue
		}
		output = append(output, current)
	}
	return os.WriteFile(cfg.RCFile, []byte(strings.Join(output, "\n")+"\n"), 0o644)
}

// EnsurePathEntryInRC appends a PATH export when a bin path is unmanaged.
func EnsurePathEntryInRC(cfg config.Config, pathEntry string) error {
	if pathEntry == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.RCFile), 0o755); err != nil {
		return err
	}
	content, _ := os.ReadFile(cfg.RCFile)
	if strings.Contains(string(content), pathEntry) {
		return nil
	}
	f, err := os.OpenFile(cfg.RCFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n# wolfpack: add managed CLIs to PATH\nexport PATH=\"%s:$PATH\"\n", pathEntry)
	if err == nil {
		fmt.Printf("Updated %s to add %s to PATH.\n", cfg.RCFile, pathEntry)
	}
	return err
}

// EnsureNVMShellInit appends nvm loading code for future shell sessions.
func EnsureNVMShellInit(cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.RCFile), 0o755); err != nil {
		return err
	}
	content, _ := os.ReadFile(cfg.RCFile)
	if strings.Contains(string(content), "nvm.sh") {
		return nil
	}
	f, err := os.OpenFile(cfg.RCFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprint(f, "\n# wolfpack: load nvm for Node.js CLIs\nexport NVM_DIR=\"$HOME/.nvm\"\n[ -s \"$NVM_DIR/nvm.sh\" ] && . \"$NVM_DIR/nvm.sh\"\n")
	if err == nil {
		fmt.Printf("Updated %s to load nvm for future shells.\n", cfg.RCFile)
	}
	return err
}

// InstallShellWrapper reloads the rc file after successful setup commands.
func InstallShellWrapper(cfg config.Config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.RCFile), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.RCFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	var output []string
	skip := false
	for _, line := range lines {
		if line == wrapperBegin {
			skip = true
			continue
		}
		if line == wrapperEnd {
			skip = false
			continue
		}
		if !skip && line != "" {
			output = append(output, line)
		}
		if !skip && line == "" && len(output) > 0 {
			output = append(output, line)
		}
	}
	block := []string{
		wrapperBegin,
		"wolfpack() {",
		"  command wolfpack \"$@\"",
		"  local status=$?",
		"  case \"${1:-}\" in",
		"    install|deps|keys)",
		fmt.Sprintf("      if [ \"$status\" -eq 0 ] && [ -f %s ]; then", ShellQuote(cfg.RCFile)),
		fmt.Sprintf("        . %s", ShellQuote(cfg.RCFile)),
		"      fi",
		"      ;;",
		"  esac",
		"  return \"$status\"",
		"}",
		wrapperEnd,
	}
	output = append(output, "", strings.Join(block, "\n"))
	if err := os.WriteFile(cfg.RCFile, []byte(strings.Join(output, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("Updated %s so wolfpack refreshes this shell after setup commands.\n", cfg.RCFile)
	return nil
}

// MaybeSourceBashrcFromShellProfile links bashrc into zsh/macOS login shells.
func MaybeSourceBashrcFromShellProfile(cfg config.Config) error {
	shellName := filepath.Base(os.Getenv("SHELL"))
	sourceLine := `[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"`
	var profile string
	var prompt string
	switch shellName {
	case "bash":
		if runtime.GOOS != "darwin" {
			return nil
		}
		profile = filepath.Join(cfg.Home, ".bash_profile")
		prompt = "On macOS, bash login shells read ~/.bash_profile. Source ~/.bashrc from it?"
	case "zsh":
		profile = filepath.Join(cfg.Home, ".zshrc")
		prompt = "Your shell appears to be zsh. Source ~/.bashrc from ~/.zshrc too?"
	default:
		return nil
	}
	content, _ := os.ReadFile(profile)
	if strings.Contains(string(content), ".bashrc") {
		return nil
	}
	if !PromptYesNo(prompt, true) {
		return nil
	}
	f, err := os.OpenFile(profile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "\n%s\n", sourceLine); err != nil {
		return err
	}
	fmt.Printf("Updated %s to source ~/.bashrc.\n", profile)
	return nil
}
