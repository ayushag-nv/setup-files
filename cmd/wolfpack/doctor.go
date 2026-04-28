package main

// doctor.go reports what Wolfpack can detect on the current machine.

import (
	"fmt"
	"runtime"
	"strings"
)

// doctor prints dependency, CLI, skill-directory, and rc-file status.
func doctor(cfg config) error {
	if err := assertSupportedOS(); err != nil {
		return err
	}
	fmt.Printf("OS: %s %s\n", runtime.GOOS, runtime.GOARCH)
	if haveCmd("curl") || haveCmd("wget") {
		fmt.Println("ok      curl or wget")
	} else {
		fmt.Println("missing curl or wget")
	}
	if commandExistsWithNVM(cfg, "node") {
		out, _ := captureShellWithNVM(cfg, "node --version")
		fmt.Printf("node: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("node: missing")
	}
	if commandExistsWithNVM(cfg, "npm") {
		version, _ := captureShellWithNVM(cfg, "npm --version")
		prefix, _ := npmGlobalPrefix(cfg)
		fmt.Printf("npm: %s\n", strings.TrimSpace(version))
		fmt.Printf("npm prefix: %s\n", prefix)
	} else {
		fmt.Println("npm: missing")
	}
	if commandExistsWithNVM(cfg, "claude") {
		out, _ := captureShellWithNVM(cfg, "claude --version")
		fmt.Printf("claude: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("claude: missing")
	}
	if commandExistsWithNVM(cfg, "codex") {
		out, _ := captureShellWithNVM(cfg, "codex --version")
		fmt.Printf("codex: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("codex: missing")
	}
	if commandExistsWithNVM(cfg, "opencode") {
		out, _ := captureShellWithNVM(cfg, "opencode --version")
		fmt.Printf("opencode: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("opencode: missing")
	}
	fmt.Printf("claude skills dir: %s\n", cfg.claudeSkillsDir)
	fmt.Printf("codex skills dir: %s\n", cfg.codexSkillsDir)
	fmt.Printf("opencode skills dir: %s\n", cfg.opencodeSkillsDir)
	fmt.Printf("api key rc file: %s\n", cfg.rcFile)
	return nil
}
