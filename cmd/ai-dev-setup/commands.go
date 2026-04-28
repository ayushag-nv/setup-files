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
	fmt.Print(`ai-dev-setup

Usage:
  ai-dev-setup install [all|claude|codex|code|skills]
  ai-dev-setup versions [claude|codex|code]
  ai-dev-setup skills [install|list]
  ai-dev-setup keys
  ai-dev-setup deps
  ai-dev-setup doctor
  ai-dev-setup help

Defaults:
  install all     Installs Claude Code, Codex CLI, skills, and prompts for API keys.
  codex/code      Both target the OpenAI Codex CLI npm package.

Environment:
  AI_SETUP_RC               Shell rc file for exported API keys (default: ~/.bashrc)
  AI_SETUP_VERSION_LIMIT    Number of npm versions to show (default: 20)
  AI_SETUP_SKILLS_REPO      ai-skills git repository URL
  AI_SETUP_SKILLS_REF       ai-skills branch/ref to install (default: main)
  AI_SETUP_SKILLS_SOURCE    Local ai-skills checkout to install from instead of fetching
  CLAUDE_SKILLS_DIR         Claude Code skills destination (default: ~/.claude/skills)
  CODEX_SKILLS_DIR          Codex skills destination (default: ${CODEX_HOME:-~/.codex}/skills)
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
ai-dev-setup
  1) Install everything
  2) Install Claude Code
  3) Install OpenAI Codex CLI
  4) Configure API keys
  5) Install skills
  6) Doctor
  7) Quit
Choose an option [1-7]: `)
		choice, _ := reader.ReadString('\n')
		switch strings.TrimSpace(choice) {
		case "1":
			return installTarget(cfg, "all")
		case "2":
			return installClaude(cfg)
		case "3":
			return installCodex(cfg)
		case "4":
			return configureKeys(cfg)
		case "5":
			return installSkills(cfg)
		case "6":
			return doctor(cfg)
		case "7", "q", "Q":
			return nil
		default:
			warn("choose a number from 1 to 7")
		}
	}
}
