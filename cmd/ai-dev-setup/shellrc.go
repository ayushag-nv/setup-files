package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ensureManagedBlock(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.rcFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if strings.Contains(string(content), beginMarker) {
		return nil
	}
	f, err := os.OpenFile(cfg.rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n%s\n", beginMarker, endMarker)
	return err
}

func upsertEnvVar(cfg config, key, value string) error {
	if err := ensureManagedBlock(cfg); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.rcFile)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("export %s=%s", key, shellQuote(value))
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
	return os.WriteFile(cfg.rcFile, []byte(strings.Join(output, "\n")+"\n"), 0o644)
}

func ensurePathEntryInRC(cfg config, pathEntry string) error {
	if pathEntry == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, _ := os.ReadFile(cfg.rcFile)
	if strings.Contains(string(content), pathEntry) {
		return nil
	}
	f, err := os.OpenFile(cfg.rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n# ai-dev-setup: add npm global CLIs to PATH\nexport PATH=\"%s:$PATH\"\n", pathEntry)
	if err == nil {
		fmt.Printf("Updated %s to add %s to PATH.\n", cfg.rcFile, pathEntry)
	}
	return err
}

func ensureNVMShellInit(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, _ := os.ReadFile(cfg.rcFile)
	if strings.Contains(string(content), "nvm.sh") {
		return nil
	}
	f, err := os.OpenFile(cfg.rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprint(f, "\n# ai-dev-setup: load nvm for Node.js CLIs\nexport NVM_DIR=\"$HOME/.nvm\"\n[ -s \"$NVM_DIR/nvm.sh\" ] && . \"$NVM_DIR/nvm.sh\"\n")
	if err == nil {
		fmt.Printf("Updated %s to load nvm for future shells.\n", cfg.rcFile)
	}
	return err
}

func installShellWrapper(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.rcFile)
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
		"ai-dev-setup() {",
		"  command ai-dev-setup \"$@\"",
		"  local status=$?",
		"  case \"${1:-}\" in",
		"    install|deps|keys)",
		fmt.Sprintf("      if [ \"$status\" -eq 0 ] && [ -f %s ]; then", shellQuote(cfg.rcFile)),
		fmt.Sprintf("        . %s", shellQuote(cfg.rcFile)),
		"      fi",
		"      ;;",
		"  esac",
		"  return \"$status\"",
		"}",
		wrapperEnd,
	}
	output = append(output, "", strings.Join(block, "\n"))
	if err := os.WriteFile(cfg.rcFile, []byte(strings.Join(output, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("Updated %s so ai-dev-setup refreshes this shell after setup commands.\n", cfg.rcFile)
	return nil
}

func maybeSourceBashrcFromShellProfile(cfg config) error {
	shellName := filepath.Base(os.Getenv("SHELL"))
	sourceLine := `[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"`
	var profile string
	var prompt string
	switch shellName {
	case "bash":
		if runtime.GOOS != "darwin" {
			return nil
		}
		profile = filepath.Join(cfg.home, ".bash_profile")
		prompt = "On macOS, bash login shells read ~/.bash_profile. Source ~/.bashrc from it?"
	case "zsh":
		profile = filepath.Join(cfg.home, ".zshrc")
		prompt = "Your shell appears to be zsh. Source ~/.bashrc from ~/.zshrc too?"
	default:
		return nil
	}
	content, _ := os.ReadFile(profile)
	if strings.Contains(string(content), ".bashrc") {
		return nil
	}
	if !promptYesNo(prompt, true) {
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
