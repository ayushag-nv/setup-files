package wolfpack

// doctor.go reports what Wolfpack can detect on the current machine.

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/tools"
)

// doctor prints dependency, CLI, skill-directory, and rc-file status.
func doctor(cfg config.Config) error {
	if err := system.AssertSupportedOS(); err != nil {
		return err
	}
	fmt.Printf("OS: %s %s\n", runtime.GOOS, runtime.GOARCH)
	if system.HaveCmd("curl") || system.HaveCmd("wget") {
		fmt.Println("ok      curl or wget")
	} else {
		fmt.Println("missing curl or wget")
	}
	if system.CommandExistsWithNVM(cfg, "node") {
		out, _ := system.CaptureShellWithNVM(cfg, "node --version")
		fmt.Printf("node: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("node: missing")
	}
	if system.CommandExistsWithNVM(cfg, "npm") {
		version, _ := system.CaptureShellWithNVM(cfg, "npm --version")
		prefix, _ := tools.NPMGlobalPrefix(cfg)
		fmt.Printf("npm: %s\n", strings.TrimSpace(version))
		fmt.Printf("npm prefix: %s\n", prefix)
	} else {
		fmt.Println("npm: missing")
	}
	if system.CommandExistsWithNVM(cfg, "claude") {
		out, _ := system.CaptureShellWithNVM(cfg, "claude --version")
		fmt.Printf("claude: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("claude: missing")
	}
	if system.CommandExistsWithNVM(cfg, "codex") {
		out, _ := system.CaptureShellWithNVM(cfg, "codex --version")
		fmt.Printf("codex: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("codex: missing")
	}
	if system.CommandExistsWithNVM(cfg, "opencode") {
		out, _ := system.CaptureShellWithNVM(cfg, "opencode --version")
		fmt.Printf("opencode: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("opencode: missing")
	}
	for _, tool := range tools.ReleaseTools {
		if system.CommandExistsWithUserBin(cfg, tool.BinaryName) {
			out, _ := system.CaptureShellWithUserBin(cfg, tool.VersionCommand)
			fmt.Printf("%s: %s\n", tool.BinaryName, strings.TrimSpace(out))
		} else {
			fmt.Printf("%s: missing\n", tool.BinaryName)
		}
	}
	fmt.Printf("wolfpack bin dir: %s\n", cfg.BinDir)
	fmt.Printf("claude skills dir: %s\n", cfg.ClaudeSkillsDir)
	fmt.Printf("codex skills dir: %s\n", cfg.CodexSkillsDir)
	fmt.Printf("opencode skills dir: %s\n", cfg.OpenCodeSkillsDir)
	fmt.Printf("api key rc file: %s\n", cfg.RCFile)
	return nil
}
