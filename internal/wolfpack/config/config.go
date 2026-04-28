package config

// Package config resolves Wolfpack runtime settings from the environment.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	defaultSkillsRef = "main"
	defaultSkillsGit = "https://github.com/ayushag-nv/ai-skills.git"
)

// Config carries resolved runtime settings used across the CLI.
type Config struct {
	Home              string
	RCFile            string
	VersionLimit      int
	SkillsRef         string
	SkillsRepo        string
	SkillsArchiveURL  string
	SkillsSource      string
	ClaudeSkillsDir   string
	CodexSkillsDir    string
	OpenCodeSkillsDir string
	BinDir            string
	NVMDir            string
}

// Load resolves defaults for rc files, skills sources, and install paths.
func Load() Config {
	home, _ := os.UserHomeDir()
	versionLimit := 20
	if raw := os.Getenv("WOLFPACK_VERSION_LIMIT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			versionLimit = parsed
		}
	}

	skillsRef := envDefault("WOLFPACK_SKILLS_REF", defaultSkillsRef)
	nvmDir := envDefault("NVM_DIR", filepath.Join(home, ".nvm"))
	codexHome := envDefault("CODEX_HOME", filepath.Join(home, ".codex"))
	archiveURL := os.Getenv("WOLFPACK_SKILLS_ARCHIVE_URL")
	if archiveURL == "" {
		archiveURL = fmt.Sprintf("https://github.com/ayushag-nv/ai-skills/archive/refs/heads/%s.tar.gz", skillsRef)
	}

	return Config{
		Home:              home,
		RCFile:            envDefault("WOLFPACK_RC", filepath.Join(home, ".bashrc")),
		VersionLimit:      versionLimit,
		SkillsRef:         skillsRef,
		SkillsRepo:        envDefault("WOLFPACK_SKILLS_REPO", defaultSkillsGit),
		SkillsArchiveURL:  archiveURL,
		SkillsSource:      os.Getenv("WOLFPACK_SKILLS_SOURCE"),
		ClaudeSkillsDir:   envDefault("CLAUDE_SKILLS_DIR", filepath.Join(home, ".claude", "skills")),
		CodexSkillsDir:    envDefault("CODEX_SKILLS_DIR", filepath.Join(codexHome, "skills")),
		OpenCodeSkillsDir: envDefault("OPENCODE_SKILLS_DIR", filepath.Join(home, ".config", "opencode", "skills")),
		BinDir:            envDefault("WOLFPACK_BIN_DIR", filepath.Join(home, ".local", "bin")),
		NVMDir:            nvmDir,
	}
}

// envDefault returns an environment value when set, otherwise the fallback.
func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
