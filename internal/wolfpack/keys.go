package wolfpack

// keys.go prompts for API keys and writes them to the configured shell rc file.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// configureKeys prompts for common provider keys and optional extra keys.
func configureKeys(cfg config.Config) error {
	fmt.Printf("API keys will be saved in %s.\n", cfg.RCFile)
	for _, key := range []string{"NVIDIA_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GITHUB_TOKEN", "GITLAB_TOKEN"} {
		if err := configureOneKey(cfg, key); err != nil {
			return err
		}
	}
	for system.PromptYesNo("Add another API key?", false) {
		name := strings.TrimSpace(system.PromptLine("Environment variable name: "))
		if name == "" {
			continue
		}
		if err := configureOneKey(cfg, name); err != nil {
			return err
		}
	}
	if err := system.InstallShellWrapper(cfg); err != nil {
		return err
	}
	if err := system.MaybeSourceBashrcFromShellProfile(cfg); err != nil {
		return err
	}
	fmt.Printf("Reload with: source %s\n", cfg.RCFile)
	return nil
}

// configureOneKey validates, prompts for, and stores a single environment key.
func configureOneKey(cfg config.Config, key string) error {
	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid environment variable name: %s", key)
	}
	value := system.PromptSecret(fmt.Sprintf("%s (leave blank to skip): ", key))
	if value == "" {
		fmt.Printf("Skipped %s.\n", key)
		return nil
	}
	if err := system.UpsertEnvVar(cfg, key, value); err != nil {
		return err
	}
	fmt.Printf("Saved %s to %s.\n", key, cfg.RCFile)
	return nil
}
