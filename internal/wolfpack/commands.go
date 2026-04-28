package wolfpack

// commands.go routes CLI arguments to the installer, diagnostics, and prompts.

import (
	"fmt"
	"os"
	"strings"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/skills"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/tools"
)

// Run loads configuration and executes a Wolfpack command.
func Run(args []string) error {
	return run(config.Load(), args)
}

// run is the top-level command dispatcher for non-interactive CLI usage.
func run(cfg config.Config, args []string) error {
	if len(args) == 0 {
		return interactiveMenu(cfg)
	}

	switch args[0] {
	case "install":
		target := "all"
		if len(args) > 1 {
			target = args[1]
		}
		return installTarget(cfg, target)
	case "versions":
		target := "all"
		if len(args) > 1 {
			target = args[1]
		}
		return listVersions(cfg, target)
	case "skills":
		subcommand := "install"
		if len(args) > 1 {
			subcommand = args[1]
		}
		switch subcommand {
		case "install":
			return skills.Install(cfg)
		case "list":
			return skills.List(cfg)
		default:
			usage()
			return fmt.Errorf("unknown skills command: %s", subcommand)
		}
	case "keys":
		return configureKeys(cfg)
	case "deps":
		return ensureDeps(cfg)
	case "doctor":
		return doctor(cfg)
	case "help", "-h", "--help":
		usage()
		return nil
	case "version", "-v", "--version":
		fmt.Printf("%s %s\n", cliName, cliVersion)
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

// usage prints the user-facing command reference.
func usage() {
	fmt.Print(`wolfpack

Usage:
  wolfpack install [all|claude|codex|code|opencode|uv|ruff|gh|glab|skills]
  wolfpack versions [claude|codex|code|opencode|uv|ruff|gh|glab]
  wolfpack skills [install|list]
  wolfpack keys
  wolfpack deps
  wolfpack doctor
  wolfpack help

Defaults:
  install all     Installs supported CLIs, skills, and prompts for API keys.
  codex/code      Both target the OpenAI Codex CLI npm package.

Environment:
  WOLFPACK_BIN_DIR          User-local binary install directory (default: ~/.local/bin)
  WOLFPACK_RC               Shell rc file for exported API keys (default: ~/.bashrc)
  WOLFPACK_VERSION_LIMIT    Number of versions to show (default: 20)
  WOLFPACK_SKILLS_REPO      ai-skills git repository URL
  WOLFPACK_SKILLS_REF       ai-skills branch/ref to install (default: main)
  WOLFPACK_SKILLS_SOURCE    Local ai-skills checkout to install from instead of fetching
  CLAUDE_SKILLS_DIR         Claude Code skills destination (default: ~/.claude/skills)
  CODEX_SKILLS_DIR          Codex skills destination (default: ${CODEX_HOME:-~/.codex}/skills)
  OPENCODE_SKILLS_DIR       OpenCode skills destination (default: ~/.config/opencode/skills)
  NVM_VERSION               nvm release tag override, such as v0.40.3
`)
}

// normalizeTarget maps aliases like "code" to the internal install target.
func normalizeTarget(target string) (string, error) {
	switch target {
	case "", "all":
		return "all", nil
	case "claude", "claude-code":
		return "claude", nil
	case "codex", "code", "openai-code", "openai-codex":
		return "codex", nil
	case "opencode", "open-code":
		return "opencode", nil
	case "uv":
		return "uv", nil
	case "ruff":
		return "ruff", nil
	case "gh", "github", "github-cli":
		return "gh", nil
	case "glab", "gitlab", "gitlab-cli":
		return "glab", nil
	case "skills", "skill":
		return "skills", nil
	default:
		return "", fmt.Errorf("unknown target: %s", target)
	}
}

// listVersions routes version listing for tools and skills.
func listVersions(cfg config.Config, target string) error {
	normalized, err := normalizeTarget(target)
	if err != nil {
		return err
	}
	if normalized == "skills" {
		return skills.List(cfg)
	}
	return tools.ListVersions(cfg, normalized)
}

// interactiveMenu provides the no-argument terminal menu.
func interactiveMenu(cfg config.Config) error {
	if !system.StdinIsTTY() {
		return installTarget(cfg, "all")
	}
	reader := system.NewInputReader()
	for {
		fmt.Fprint(os.Stderr, `
wolfpack
  1) Install everything
  2) Install Claude Code
  3) Install OpenAI Codex CLI
  4) Install OpenCode CLI
  5) Install uv
  6) Install Ruff
  7) Install GitHub CLI
  8) Install GitLab CLI
  9) Configure API keys
  10) Install skills
  11) Doctor
  12) Quit
Choose an option [1-12]: `)
		choice, _ := reader.ReadString('\n')
		switch strings.TrimSpace(choice) {
		case "1":
			return installTarget(cfg, "all")
		case "2":
			return installClaude(cfg)
		case "3":
			return installCodex(cfg)
		case "4":
			return installOpenCode(cfg)
		case "5":
			return tools.InstallUV(cfg)
		case "6":
			return tools.InstallRuff(cfg)
		case "7":
			return tools.InstallGH(cfg)
		case "8":
			return tools.InstallGLab(cfg)
		case "9":
			return configureKeys(cfg)
		case "10":
			return skills.Install(cfg)
		case "11":
			return doctor(cfg)
		case "12", "q", "Q":
			return nil
		default:
			system.Warn("choose a number from 1 to 12")
		}
	}
}
