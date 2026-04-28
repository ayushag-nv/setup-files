package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type config struct {
	home              string
	rcFile            string
	versionLimit      int
	skillsRef         string
	skillsRepo        string
	skillsArchiveURL  string
	skillsSource      string
	claudeSkillsDir   string
	codexSkillsDir    string
	opencodeSkillsDir string
	nvmDir            string
}

func loadConfig() config {
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

	return config{
		home:              home,
		rcFile:            envDefault("WOLFPACK_RC", filepath.Join(home, ".bashrc")),
		versionLimit:      versionLimit,
		skillsRef:         skillsRef,
		skillsRepo:        envDefault("WOLFPACK_SKILLS_REPO", defaultSkillsGit),
		skillsArchiveURL:  archiveURL,
		skillsSource:      os.Getenv("WOLFPACK_SKILLS_SOURCE"),
		claudeSkillsDir:   envDefault("CLAUDE_SKILLS_DIR", filepath.Join(home, ".claude", "skills")),
		codexSkillsDir:    envDefault("CODEX_SKILLS_DIR", filepath.Join(codexHome, "skills")),
		opencodeSkillsDir: envDefault("OPENCODE_SKILLS_DIR", filepath.Join(home, ".config", "opencode", "skills")),
		nvmDir:            nvmDir,
	}
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
