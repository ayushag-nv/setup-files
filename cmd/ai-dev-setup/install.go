package main

import "fmt"

func installTarget(cfg config, target string) error {
	normalized, err := normalizeTarget(target)
	if err != nil {
		return err
	}

	switch normalized {
	case "all":
		if err := installClaude(cfg); err != nil {
			return err
		}
		if err := installCodex(cfg); err != nil {
			return err
		}
		if err := installSkills(cfg); err != nil {
			return err
		}
		return configureKeys(cfg)
	case "claude":
		return installClaude(cfg)
	case "codex":
		return installCodex(cfg)
	case "skills":
		return installSkills(cfg)
	default:
		return fmt.Errorf("unknown target: %s", target)
	}
}

func installClaude(cfg config) error {
	return installNPMTool(cfg, "Claude Code", claudePackage, "claude")
}

func installCodex(cfg config) error {
	return installNPMTool(cfg, "OpenAI Codex CLI", codexPackage, "codex")
}

func ensureDeps(cfg config) error {
	if err := assertSupportedOS(); err != nil {
		return err
	}
	if !haveCmd("git") {
		warn("git is unavailable; skills install will fall back to archive downloads")
	}
	if err := ensureNode(cfg); err != nil {
		return err
	}
	if err := ensureNPMCLIPath(cfg); err != nil {
		return err
	}
	if err := installShellWrapper(cfg); err != nil {
		return err
	}
	if err := maybeSourceBashrcFromShellProfile(cfg); err != nil {
		return err
	}
	fmt.Println("Dependencies are ready.")
	fmt.Printf("For this terminal, run once: source %s\n", cfg.rcFile)
	return nil
}
