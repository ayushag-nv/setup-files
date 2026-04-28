package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func run(cfg config, args []string) error {
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
			return installSkills(cfg)
		case "list":
			return listSkills(cfg)
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

func usage() {
	fmt.Print(`wolfpack

Usage:
  wolfpack install [all|claude|codex|code|opencode|skills]
  wolfpack versions [claude|codex|code|opencode]
  wolfpack skills [install|list]
  wolfpack keys
  wolfpack deps
  wolfpack doctor
  wolfpack help

Defaults:
  install all     Installs Claude Code, Codex CLI, OpenCode, skills, and prompts for API keys.
  codex/code      Both target the OpenAI Codex CLI npm package.

Environment:
  WOLFPACK_RC               Shell rc file for exported API keys (default: ~/.bashrc)
  WOLFPACK_VERSION_LIMIT    Number of npm versions to show (default: 20)
  WOLFPACK_SKILLS_REPO      ai-skills git repository URL
  WOLFPACK_SKILLS_REF       ai-skills branch/ref to install (default: main)
  WOLFPACK_SKILLS_SOURCE    Local ai-skills checkout to install from instead of fetching
  CLAUDE_SKILLS_DIR         Claude Code skills destination (default: ~/.claude/skills)
  CODEX_SKILLS_DIR          Codex skills destination (default: ${CODEX_HOME:-~/.codex}/skills)
  OPENCODE_SKILLS_DIR       OpenCode skills destination (default: ~/.config/opencode/skills)
  NVM_VERSION               nvm release tag override, such as v0.40.3
`)
}

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
	case "skills", "skill":
		return "skills", nil
	default:
		return "", fmt.Errorf("unknown target: %s", target)
	}
}

func assertSupportedOS() error {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		return nil
	}
	return fmt.Errorf("unsupported OS: %s. This CLI supports macOS and Linux", runtime.GOOS)
}

func interactiveMenu(cfg config) error {
	if !stdinIsTTY() {
		return installTarget(cfg, "all")
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, `
wolfpack
  1) Install everything
  2) Install Claude Code
  3) Install OpenAI Codex CLI
  4) Install OpenCode CLI
  5) Configure API keys
  6) Install skills
  7) Doctor
  8) Quit
Choose an option [1-8]: `)
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
			return configureKeys(cfg)
		case "6":
			return installSkills(cfg)
		case "7":
			return doctor(cfg)
		case "8", "q", "Q":
			return nil
		default:
			warn("choose a number from 1 to 8")
		}
	}
}
