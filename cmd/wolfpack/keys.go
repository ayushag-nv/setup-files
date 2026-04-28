package main

// keys.go prompts for API keys and writes them to the configured shell rc file.

import (
	"fmt"
	"regexp"
	"strings"
)

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// configureKeys prompts for common provider keys and optional extra keys.
func configureKeys(cfg config) error {
	fmt.Printf("API keys will be saved in %s.\n", cfg.rcFile)
	for _, key := range []string{"NVIDIA_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		if err := configureOneKey(cfg, key); err != nil {
			return err
		}
	}
	for promptYesNo("Add another API key?", false) {
		name := strings.TrimSpace(promptLine("Environment variable name: "))
		if name == "" {
			continue
		}
		if err := configureOneKey(cfg, name); err != nil {
			return err
		}
	}
	if err := installShellWrapper(cfg); err != nil {
		return err
	}
	if err := maybeSourceBashrcFromShellProfile(cfg); err != nil {
		return err
	}
	fmt.Printf("Reload with: source %s\n", cfg.rcFile)
	return nil
}

// configureOneKey validates, prompts for, and stores a single environment key.
func configureOneKey(cfg config, key string) error {
	if !envKeyPattern.MatchString(key) {
		return fmt.Errorf("invalid environment variable name: %s", key)
	}
	value := promptSecret(fmt.Sprintf("%s (leave blank to skip): ", key))
	if value == "" {
		fmt.Printf("Skipped %s.\n", key)
		return nil
	}
	if err := upsertEnvVar(cfg, key, value); err != nil {
		return err
	}
	fmt.Printf("Saved %s to %s.\n", key, cfg.rcFile)
	return nil
}
