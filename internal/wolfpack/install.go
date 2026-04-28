package wolfpack

// install.go maps install targets to tools, skills, and setup steps.

import (
	"fmt"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/skills"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/tools"
)

type dependencyInstaller struct {
	target string
	label  string
	run    func(config.Config) error
}

var developerDependencyInstallers = []dependencyInstaller{
	{target: "uv", label: "uv", run: tools.InstallUV},
	{target: "ruff", label: "Ruff", run: tools.InstallRuff},
	{target: "gh", label: "GitHub CLI", run: tools.InstallGH},
	{target: "glab", label: "GitLab CLI", run: tools.InstallGLab},
}

// installTarget runs the requested target, including the full "all" flow.
func installTarget(cfg config.Config, target string) error {
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
		if err := installDeveloperDependencies(cfg); err != nil {
			return err
		}
		if err := skills.Install(cfg); err != nil {
			return err
		}
		return configureKeys(cfg)
	case "claude":
		return installClaude(cfg)
	case "codex":
		return installCodex(cfg)
	case "opencode":
		return installOpenCode(cfg)
	case "uv":
		return tools.InstallUV(cfg)
	case "ruff":
		return tools.InstallRuff(cfg)
	case "gh":
		return tools.InstallGH(cfg)
	case "glab":
		return tools.InstallGLab(cfg)
	case "skills":
		return skills.Install(cfg)
	default:
		return fmt.Errorf("unknown target: %s", target)
	}
}

// installClaude installs the Claude Code npm package.
func installClaude(cfg config.Config) error {
	return tools.InstallNPMTarget(cfg, "claude")
}

// installCodex installs the OpenAI Codex CLI npm package.
func installCodex(cfg config.Config) error {
	return tools.InstallNPMTarget(cfg, "codex")
}

// installOpenCode installs the OpenCode CLI npm package.
func installOpenCode(cfg config.Config) error {
	return tools.InstallNPMTarget(cfg, "opencode")
}

// installDeveloperDependencies installs shared non-AI developer tools.
func installDeveloperDependencies(cfg config.Config) error {
	for _, installer := range developerDependencyInstallers {
		if err := installer.run(cfg); err != nil {
			return fmt.Errorf("install %s: %w", installer.label, err)
		}
	}
	return nil
}

// ensureDeps prepares shared dependencies and developer tools without installing AI CLIs.
func ensureDeps(cfg config.Config) error {
	if err := system.AssertSupportedOS(); err != nil {
		return err
	}
	if !system.HaveCmd("git") {
		system.Warn("git is unavailable; skills install will fall back to archive downloads")
	}
	if err := system.EnsurePathEntryInRC(cfg, cfg.BinDir); err != nil {
		return err
	}
	if err := tools.EnsureNode(cfg); err != nil {
		return err
	}
	if err := tools.EnsureNPMCLIPath(cfg); err != nil {
		return err
	}
	if err := installDeveloperDependencies(cfg); err != nil {
		return err
	}
	if err := system.InstallShellWrapper(cfg); err != nil {
		return err
	}
	if err := system.MaybeSourceBashrcFromShellProfile(cfg); err != nil {
		return err
	}
	fmt.Println("Dependencies are ready.")
	fmt.Printf("For this terminal, run once: source %s\n", cfg.RCFile)
	return nil
}
