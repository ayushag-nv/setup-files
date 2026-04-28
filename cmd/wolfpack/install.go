package main

// install.go maps install targets to npm-backed tools and setup steps.

import "fmt"

// npmTool describes one global npm package and the command it should expose.
type npmTool struct {
	target      string
	label       string
	packageName string
	binaryName  string
}

// npmTools is the list of AI coding CLIs Wolfpack can install through npm.
var npmTools = []npmTool{
	{target: "claude", label: "Claude Code", packageName: claudePackage, binaryName: "claude"},
	{target: "codex", label: "OpenAI Codex CLI", packageName: codexPackage, binaryName: "codex"},
	{target: "opencode", label: "OpenCode CLI", packageName: opencodePackage, binaryName: "opencode"},
}

// npmToolByTarget returns metadata for a normalized npm-backed install target.
func npmToolByTarget(target string) (npmTool, bool) {
	for _, tool := range npmTools {
		if tool.target == target {
			return tool, true
		}
	}
	return npmTool{}, false
}

// installTarget runs the requested target, including the full "all" flow.
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
		if err := installOpenCode(cfg); err != nil {
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
	case "opencode":
		return installOpenCode(cfg)
	case "skills":
		return installSkills(cfg)
	default:
		return fmt.Errorf("unknown target: %s", target)
	}
}

// installClaude installs the Claude Code npm package.
func installClaude(cfg config) error {
	return installNPMTarget(cfg, "claude")
}

// installCodex installs the OpenAI Codex CLI npm package.
func installCodex(cfg config) error {
	return installNPMTarget(cfg, "codex")
}

// installOpenCode installs the OpenCode CLI npm package.
func installOpenCode(cfg config) error {
	return installNPMTarget(cfg, "opencode")
}

// installNPMTarget looks up target metadata and delegates to the npm installer.
func installNPMTarget(cfg config, target string) error {
	tool, ok := npmToolByTarget(target)
	if !ok {
		return fmt.Errorf("unknown npm target: %s", target)
	}
	return installNPMTool(cfg, tool.label, tool.packageName, tool.binaryName)
}

// ensureDeps prepares shared dependencies without installing AI CLIs.
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
