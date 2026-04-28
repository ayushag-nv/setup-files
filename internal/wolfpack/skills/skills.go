package skills

// Package skills installs shared SKILL.md bundles for supported agents.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
)

// isExcludedSkill filters out skills Wolfpack must not install.
func isExcludedSkill(name string) bool {
	return name == "ultimate-auto" || name == "ultimate-bypass"
}

// prepareSkillsSource resolves a local, git, or archive source for skills.
func prepareSkillsSource(cfg config.Config, tmpDir string) (string, error) {
	if cfg.SkillsSource != "" {
		if stat, err := os.Stat(filepath.Join(cfg.SkillsSource, "skills")); err != nil || !stat.IsDir() {
			return "", fmt.Errorf("WOLFPACK_SKILLS_SOURCE must contain a skills/ directory: %s", cfg.SkillsSource)
		}
		return cfg.SkillsSource, nil
	}

	sourceDir := filepath.Join(tmpDir, "source")
	if system.HaveCmd("git") {
		fmt.Fprintf(os.Stderr, "Cloning skills from %s (%s).\n", cfg.SkillsRepo, cfg.SkillsRef)
		if err := system.RunCommandQuiet("git", "clone", "--depth", "1", "--branch", cfg.SkillsRef, cfg.SkillsRepo, sourceDir); err == nil {
			if stat, err := os.Stat(filepath.Join(sourceDir, "skills")); err == nil && stat.IsDir() {
				return sourceDir, nil
			}
			return "", errors.New("cloned skills repository does not contain a skills/ directory")
		}
		system.Warn("git clone failed; falling back to archive download")
	}

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return "", err
	}
	archivePath := filepath.Join(tmpDir, "ai-skills.tar.gz")
	fmt.Fprintf(os.Stderr, "Downloading skills from %s.\n", cfg.SkillsArchiveURL)
	if err := system.DownloadFile(cfg.SkillsArchiveURL, archivePath); err != nil {
		return "", err
	}
	if err := system.ExtractTarGZStripFirstComponent(archivePath, sourceDir); err != nil {
		return "", err
	}
	if stat, err := os.Stat(filepath.Join(sourceDir, "skills")); err != nil || !stat.IsDir() {
		return "", errors.New("downloaded skills archive does not contain a skills/ directory")
	}
	return sourceDir, nil
}

// List prints installable skill names from the resolved source.
func List(cfg config.Config) error {
	tmpDir, err := os.MkdirTemp("", "wolfpack-skills-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	sourceDir, err := prepareSkillsSource(cfg, tmpDir)
	if err != nil {
		return err
	}
	skills, err := availableSkills(sourceDir)
	if err != nil {
		return err
	}
	for _, skill := range skills {
		fmt.Println(skill)
	}
	return nil
}

// Install copies shared skills into Claude, Codex, and OpenCode paths.
func Install(cfg config.Config) error {
	tmpDir, err := os.MkdirTemp("", "wolfpack-skills-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	sourceDir, err := prepareSkillsSource(cfg, tmpDir)
	if err != nil {
		return err
	}
	if err := installSkillsToDir(sourceDir, "Claude Code", cfg.ClaudeSkillsDir); err != nil {
		return err
	}
	if err := installSkillsToDir(sourceDir, "Codex", cfg.CodexSkillsDir); err != nil {
		return err
	}
	return installSkillsToDir(sourceDir, "OpenCode", cfg.OpenCodeSkillsDir)
}

// availableSkills returns sorted skills that contain a SKILL.md and are allowed.
func availableSkills(sourceDir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(sourceDir, "skills"))
	if err != nil {
		return nil, err
	}
	var skills []string
	for _, entry := range entries {
		if !entry.IsDir() || isExcludedSkill(entry.Name()) {
			continue
		}
		if _, err := os.Stat(filepath.Join(sourceDir, "skills", entry.Name(), "SKILL.md")); err == nil {
			skills = append(skills, entry.Name())
		}
	}
	sort.Strings(skills)
	return skills, nil
}

// installSkillsToDir copies all available skills into one agent destination.
func installSkillsToDir(sourceDir, targetLabel, targetDir string) error {
	skills, err := availableSkills(sourceDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	for _, skill := range skills {
		src := filepath.Join(sourceDir, "skills", skill)
		dst := filepath.Join(targetDir, skill)
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
		if err := system.CopyTree(src, dst); err != nil {
			return err
		}
	}
	fmt.Printf("Installed %d skills for %s into %s.\n", len(skills), targetLabel, targetDir)
	return nil
}
