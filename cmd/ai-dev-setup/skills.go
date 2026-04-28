package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func isExcludedSkill(name string) bool {
	return name == "ultimate-auto" || name == "ultimate-bypass"
}

func prepareSkillsSource(cfg config, tmpDir string) (string, error) {
	if cfg.skillsSource != "" {
		if stat, err := os.Stat(filepath.Join(cfg.skillsSource, "skills")); err != nil || !stat.IsDir() {
			return "", fmt.Errorf("AI_SETUP_SKILLS_SOURCE must contain a skills/ directory: %s", cfg.skillsSource)
		}
		return cfg.skillsSource, nil
	}

	sourceDir := filepath.Join(tmpDir, "source")
	if haveCmd("git") {
		fmt.Fprintf(os.Stderr, "Cloning skills from %s (%s).\n", cfg.skillsRepo, cfg.skillsRef)
		if err := runCommandQuiet("git", "clone", "--depth", "1", "--branch", cfg.skillsRef, cfg.skillsRepo, sourceDir); err == nil {
			if stat, err := os.Stat(filepath.Join(sourceDir, "skills")); err == nil && stat.IsDir() {
				return sourceDir, nil
			}
			return "", errors.New("cloned skills repository does not contain a skills/ directory")
		}
		warn("git clone failed; falling back to archive download")
	}

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return "", err
	}
	archivePath := filepath.Join(tmpDir, "ai-skills.tar.gz")
	fmt.Fprintf(os.Stderr, "Downloading skills from %s.\n", cfg.skillsArchiveURL)
	if err := downloadFile(cfg.skillsArchiveURL, archivePath); err != nil {
		return "", err
	}
	if err := extractTarGZStripFirstComponent(archivePath, sourceDir); err != nil {
		return "", err
	}
	if stat, err := os.Stat(filepath.Join(sourceDir, "skills")); err != nil || !stat.IsDir() {
		return "", errors.New("downloaded skills archive does not contain a skills/ directory")
	}
	return sourceDir, nil
}

func listSkills(cfg config) error {
	if err := assertSupportedOS(); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "ai-dev-setup-skills-*")
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

func installSkills(cfg config) error {
	if err := assertSupportedOS(); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "ai-dev-setup-skills-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	sourceDir, err := prepareSkillsSource(cfg, tmpDir)
	if err != nil {
		return err
	}
	if err := installSkillsToDir(sourceDir, "Claude Code", cfg.claudeSkillsDir); err != nil {
		return err
	}
	return installSkillsToDir(sourceDir, "Codex", cfg.codexSkillsDir)
}

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
		if err := copyTree(src, dst); err != nil {
			return err
		}
	}
	fmt.Printf("Installed %d skills for %s into %s.\n", len(skills), targetLabel, targetDir)
	return nil
}
