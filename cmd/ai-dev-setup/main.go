package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

const (
	cliName          = "ai-dev-setup"
	cliVersion       = "0.3.0"
	claudePackage    = "@anthropic-ai/claude-code"
	codexPackage     = "@openai/codex"
	minNodeMajor     = 18
	beginMarker      = "# >>> ai-dev-setup managed env >>>"
	endMarker        = "# <<< ai-dev-setup managed env <<<"
	wrapperBegin     = "# >>> ai-dev-setup shell wrapper >>>"
	wrapperEnd       = "# <<< ai-dev-setup shell wrapper <<<"
	defaultSkillsRef = "main"
	defaultSkillsGit = "https://github.com/ayushag-nv/ai-skills.git"
)

type config struct {
	home             string
	rcFile           string
	versionLimit     int
	skillsRef        string
	skillsRepo       string
	skillsArchiveURL string
	skillsSource     string
	claudeSkillsDir  string
	codexSkillsDir   string
	nvmDir           string
}

type npmMetadata struct {
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]json.RawMessage `json:"versions"`
}

type semver struct {
	raw        string
	major      int
	minor      int
	patch      int
	prerelease string
}

var semverPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)

func main() {
	cfg := loadConfig()
	if err := run(cfg, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func loadConfig() config {
	home, _ := os.UserHomeDir()
	versionLimit := 20
	if raw := os.Getenv("AI_SETUP_VERSION_LIMIT"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			versionLimit = parsed
		}
	}

	skillsRef := envDefault("AI_SETUP_SKILLS_REF", defaultSkillsRef)
	nvmDir := envDefault("NVM_DIR", filepath.Join(home, ".nvm"))
	codexHome := envDefault("CODEX_HOME", filepath.Join(home, ".codex"))
	archiveURL := os.Getenv("AI_SETUP_SKILLS_ARCHIVE_URL")
	if archiveURL == "" {
		archiveURL = fmt.Sprintf("https://github.com/ayushag-nv/ai-skills/archive/refs/heads/%s.tar.gz", skillsRef)
	}

	return config{
		home:             home,
		rcFile:           envDefault("AI_SETUP_RC", filepath.Join(home, ".bashrc")),
		versionLimit:     versionLimit,
		skillsRef:        skillsRef,
		skillsRepo:       envDefault("AI_SETUP_SKILLS_REPO", defaultSkillsGit),
		skillsArchiveURL: archiveURL,
		skillsSource:     os.Getenv("AI_SETUP_SKILLS_SOURCE"),
		claudeSkillsDir:  envDefault("CLAUDE_SKILLS_DIR", filepath.Join(home, ".claude", "skills")),
		codexSkillsDir:   envDefault("CODEX_SKILLS_DIR", filepath.Join(codexHome, "skills")),
		nvmDir:           nvmDir,
	}
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

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

func installNPMTool(cfg config, label, packageName, binaryName string) error {
	if err := assertSupportedOS(); err != nil {
		return err
	}
	if err := ensureNode(cfg); err != nil {
		return err
	}
	if err := ensureNPMCLIPath(cfg); err != nil {
		return err
	}

	version, err := choosePackageVersion(cfg, label, packageName)
	if err != nil {
		return err
	}
	spec := packageName + "@" + version
	fmt.Printf("Installing %s with npm package %s.\n", label, spec)
	if err := runShellWithNVM(cfg, "npm install -g "+shellQuote(spec)); err != nil {
		return err
	}
	if err := ensureNPMCLIPath(cfg); err != nil {
		return err
	}

	if commandExistsWithNVM(cfg, binaryName) {
		fmt.Printf("%s installed:\n", label)
		_ = runShellWithNVM(cfg, binaryName+" --version")
	} else {
		warn("%s install completed, but %q is not on PATH in this shell", label, binaryName)
	}

	if err := installShellWrapper(cfg); err != nil {
		return err
	}
	if err := maybeSourceBashrcFromShellProfile(cfg); err != nil {
		return err
	}
	fmt.Printf("For this terminal, run once: source %s\n", cfg.rcFile)
	return nil
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

func ensureNode(cfg config) error {
	if commandExistsWithNVM(cfg, "node") && commandExistsWithNVM(cfg, "npm") {
		major, err := nodeMajor(cfg)
		if err == nil && major >= minNodeMajor && npmGlobalPrefixWritable(cfg) {
			return nil
		}
		if err == nil && major < minNodeMajor {
			warn("Node.js %d+ is required; current major is %d", minNodeMajor, major)
		} else if !npmGlobalPrefixWritable(cfg) {
			warn("npm global prefix is not writable; installing user-local Node.js through nvm")
		}
	} else {
		fmt.Println("Node.js and npm are required; installing them through nvm.")
	}

	if err := installNVMAndNode(cfg); err != nil {
		return err
	}
	if !commandExistsWithNVM(cfg, "node") {
		return errors.New("node was not found after installation")
	}
	if !commandExistsWithNVM(cfg, "npm") {
		return errors.New("npm was not found after installation")
	}
	return nil
}

func installNVMAndNode(cfg config) error {
	if !commandExistsWithNVM(cfg, "nvm") {
		version := os.Getenv("NVM_VERSION")
		if version == "" {
			latest, err := latestNVMVersion()
			if err == nil {
				version = latest
			}
		}
		if version == "" {
			version = "v0.40.3"
			warn("could not detect latest nvm release; falling back to %s", version)
		}

		fmt.Printf("Installing nvm %s.\n", version)
		tmp, err := os.CreateTemp("", "ai-dev-setup-nvm-*.sh")
		if err != nil {
			return err
		}
		tmp.Close()
		defer os.Remove(tmp.Name())

		url := fmt.Sprintf("https://raw.githubusercontent.com/nvm-sh/nvm/%s/install.sh", version)
		if err := downloadFile(url, tmp.Name()); err != nil {
			return err
		}
		if err := runCommand("bash", tmp.Name()); err != nil {
			return err
		}
	}

	fmt.Println("Installing latest Node.js LTS through nvm.")
	if err := runShellWithNVM(cfg, "nvm install --lts && nvm alias default 'lts/*' >/dev/null && nvm use --lts >/dev/null"); err != nil {
		return err
	}
	return ensureNVMShellInit(cfg)
}

func latestNVMVersion() (string, error) {
	resp, err := http.Get("https://api.github.com/repos/nvm-sh/nvm/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.TagName, nil
}

func nodeMajor(cfg config) (int, error) {
	out, err := captureShellWithNVM(cfg, "node -p \"process.versions.node.split('.')[0]\"")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

func npmGlobalPrefix(cfg config) (string, error) {
	out, err := captureShellWithNVM(cfg, "npm config get prefix")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func npmGlobalPrefixWritable(cfg config) bool {
	prefix, err := npmGlobalPrefix(cfg)
	if err != nil || prefix == "" {
		return false
	}
	info, err := os.Stat(prefix)
	if err == nil && info.IsDir() {
		return canWriteDir(prefix)
	}
	parent := filepath.Dir(prefix)
	if parent == "." || parent == prefix {
		return false
	}
	return canWriteDir(parent)
}

func canWriteDir(dir string) bool {
	tmp, err := os.CreateTemp(dir, ".ai-dev-setup-*")
	if err != nil {
		return false
	}
	tmp.Close()
	os.Remove(tmp.Name())
	return true
}

func ensureNPMCLIPath(cfg config) error {
	if !commandExistsWithNVM(cfg, "npm") {
		return nil
	}
	prefix, err := npmGlobalPrefix(cfg)
	if err != nil || prefix == "" {
		return err
	}
	binDir := filepath.Join(prefix, "bin")
	if strings.HasPrefix(binDir, cfg.nvmDir+string(os.PathSeparator)) {
		return ensureNVMShellInit(cfg)
	}
	return ensurePathEntryInRC(cfg, binDir)
}

func npmVersionsForPackage(packageName string, limit int) (string, []string, error) {
	url := "https://registry.npmjs.org/" + strings.ReplaceAll(packageName, "/", "%2F")
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("npm registry returned HTTP %d", resp.StatusCode)
	}

	var metadata npmMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", nil, err
	}
	latest := metadata.DistTags["latest"]
	var versions []string
	for version := range metadata.Versions {
		if parseSemver(version) != nil {
			versions = append(versions, version)
		}
	}
	stable := versions[:0]
	for _, version := range versions {
		if regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(version) {
			stable = append(stable, version)
		}
	}
	if len(stable) > 0 {
		versions = stable
	}
	sort.Slice(versions, func(i, j int) bool {
		return compareSemver(versions[i], versions[j]) < 0
	})
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}
	if len(versions) > limit {
		versions = versions[:limit]
	}
	return latest, versions, nil
}

func parseSemver(version string) *semver {
	match := semverPattern.FindStringSubmatch(version)
	if match == nil {
		return nil
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	return &semver{raw: version, major: major, minor: minor, patch: patch, prerelease: match[4]}
}

func compareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	if pa == nil && pb == nil {
		return strings.Compare(a, b)
	}
	if pa == nil {
		return -1
	}
	if pb == nil {
		return 1
	}
	if pa.major != pb.major {
		return pa.major - pb.major
	}
	if pa.minor != pb.minor {
		return pa.minor - pb.minor
	}
	if pa.patch != pb.patch {
		return pa.patch - pb.patch
	}
	if pa.prerelease == pb.prerelease {
		return 0
	}
	if pa.prerelease == "" {
		return 1
	}
	if pb.prerelease == "" {
		return -1
	}
	return strings.Compare(pa.prerelease, pb.prerelease)
}

func choosePackageVersion(cfg config, label, packageName string) (string, error) {
	fmt.Fprintf(os.Stderr, "Fetching available %s versions from npm registry.\n", label)
	latest, versions, err := npmVersionsForPackage(packageName, cfg.versionLimit)
	if err != nil {
		return "", err
	}
	if latest == "" {
		return "", fmt.Errorf("could not determine latest version for %s", packageName)
	}
	if !stdinIsTTY() {
		return "latest", nil
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s versions:\n", label)
	fmt.Fprintf(os.Stderr, "  0) latest (%s)\n", latest)
	for i, version := range versions {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, version)
	}
	fmt.Fprintf(os.Stderr, "  or enter an exact version, for example %s\n", latest)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "Choose %s version [0-%d or exact] (default 0): ", label, len(versions))
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		if choice == "" || choice == "0" {
			return "latest", nil
		}
		if semverPattern.MatchString(choice) {
			return choice, nil
		}
		index, err := strconv.Atoi(choice)
		if err == nil && index >= 1 && index <= len(versions) {
			return versions[index-1], nil
		}
		warn("enter a number between 0 and %d, or an exact version", len(versions))
	}
}

func listVersions(cfg config, target string) error {
	normalized, err := normalizeTarget(target)
	if err != nil {
		return err
	}
	switch normalized {
	case "all":
		if err := listVersions(cfg, "claude"); err != nil {
			return err
		}
		fmt.Println()
		return listVersions(cfg, "codex")
	case "skills":
		return listSkills(cfg)
	}

	if err := ensureNode(cfg); err != nil {
		return err
	}
	var label, packageName string
	if normalized == "claude" {
		label = "Claude Code"
		packageName = claudePackage
	} else {
		label = "OpenAI Codex CLI"
		packageName = codexPackage
	}
	latest, versions, err := npmVersionsForPackage(packageName, cfg.versionLimit)
	if err != nil {
		return err
	}
	fmt.Printf("%s npm versions:\n", label)
	fmt.Printf("latest: %s\n", latest)
	for _, version := range versions {
		fmt.Printf("  %s\n", version)
	}
	return nil
}

func configureKeys(cfg config) error {
	fmt.Printf("API keys will be saved in %s.\n", cfg.rcFile)
	for _, key := range []string{"NVIDIA_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		if err := configureOneKey(cfg, key); err != nil {
			return err
		}
	}
	for promptYesNo("Add another API key?", false) {
		name := promptLine("Environment variable name: ")
		name = strings.TrimSpace(name)
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

func configureOneKey(cfg config, key string) error {
	if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(key) {
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

func promptLine(prompt string) string {
	if stdinIsTTY() {
		fmt.Fprint(os.Stderr, prompt)
	}
	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

func promptSecret(prompt string) string {
	if !stdinIsTTY() {
		return promptLine("")
	}
	fmt.Fprint(os.Stderr, prompt)
	if haveCmd("stty") {
		_ = runCommandWithIO("stty", []string{"-echo"}, os.Stdin, io.Discard, io.Discard)
		defer func() {
			_ = runCommandWithIO("stty", []string{"echo"}, os.Stdin, io.Discard, io.Discard)
			fmt.Fprintln(os.Stderr)
		}()
	}
	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	return strings.TrimRight(value, "\r\n")
}

func promptYesNo(prompt string, defaultYes bool) bool {
	if !stdinIsTTY() {
		return defaultYes
	}
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "%s %s ", prompt, suffix)
		answer, _ := reader.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "" {
			return defaultYes
		}
		if answer == "y" || answer == "yes" {
			return true
		}
		if answer == "n" || answer == "no" {
			return false
		}
		warn("answer yes or no")
	}
}

func ensureManagedBlock(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.rcFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if strings.Contains(string(content), beginMarker) {
		return nil
	}
	f, err := os.OpenFile(cfg.rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n%s\n", beginMarker, endMarker)
	return err
}

func upsertEnvVar(cfg config, key, value string) error {
	if err := ensureManagedBlock(cfg); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.rcFile)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("export %s=%s", key, shellQuote(value))
	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	var output []string
	inBlock := false
	seen := false
	for _, current := range lines {
		switch current {
		case beginMarker:
			inBlock = true
			output = append(output, current)
			continue
		case endMarker:
			if !seen {
				output = append(output, line)
			}
			inBlock = false
			output = append(output, current)
			continue
		}
		if inBlock && strings.HasPrefix(current, "export "+key+"=") {
			if !seen {
				output = append(output, line)
				seen = true
			}
			continue
		}
		output = append(output, current)
	}
	return os.WriteFile(cfg.rcFile, []byte(strings.Join(output, "\n")+"\n"), 0o644)
}

func ensurePathEntryInRC(cfg config, pathEntry string) error {
	if pathEntry == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, _ := os.ReadFile(cfg.rcFile)
	if strings.Contains(string(content), pathEntry) {
		return nil
	}
	f, err := os.OpenFile(cfg.rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n# ai-dev-setup: add npm global CLIs to PATH\nexport PATH=\"%s:$PATH\"\n", pathEntry)
	if err == nil {
		fmt.Printf("Updated %s to add %s to PATH.\n", cfg.rcFile, pathEntry)
	}
	return err
}

func ensureNVMShellInit(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, _ := os.ReadFile(cfg.rcFile)
	if strings.Contains(string(content), "nvm.sh") {
		return nil
	}
	f, err := os.OpenFile(cfg.rcFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprint(f, "\n# ai-dev-setup: load nvm for Node.js CLIs\nexport NVM_DIR=\"$HOME/.nvm\"\n[ -s \"$NVM_DIR/nvm.sh\" ] && . \"$NVM_DIR/nvm.sh\"\n")
	if err == nil {
		fmt.Printf("Updated %s to load nvm for future shells.\n", cfg.rcFile)
	}
	return err
}

func installShellWrapper(cfg config) error {
	if err := os.MkdirAll(filepath.Dir(cfg.rcFile), 0o755); err != nil {
		return err
	}
	content, err := os.ReadFile(cfg.rcFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(content), "\n"), "\n")
	var output []string
	skip := false
	for _, line := range lines {
		if line == wrapperBegin {
			skip = true
			continue
		}
		if line == wrapperEnd {
			skip = false
			continue
		}
		if !skip && line != "" {
			output = append(output, line)
		}
		if !skip && line == "" && len(output) > 0 {
			output = append(output, line)
		}
	}
	block := []string{
		wrapperBegin,
		"ai-dev-setup() {",
		"  command ai-dev-setup \"$@\"",
		"  local status=$?",
		"  case \"${1:-}\" in",
		"    install|deps|keys)",
		fmt.Sprintf("      if [ \"$status\" -eq 0 ] && [ -f %s ]; then", shellQuote(cfg.rcFile)),
		fmt.Sprintf("        . %s", shellQuote(cfg.rcFile)),
		"      fi",
		"      ;;",
		"  esac",
		"  return \"$status\"",
		"}",
		wrapperEnd,
	}
	output = append(output, "", strings.Join(block, "\n"))
	if err := os.WriteFile(cfg.rcFile, []byte(strings.Join(output, "\n")+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("Updated %s so ai-dev-setup refreshes this shell after setup commands.\n", cfg.rcFile)
	return nil
}

func maybeSourceBashrcFromShellProfile(cfg config) error {
	shellName := filepath.Base(os.Getenv("SHELL"))
	sourceLine := `[ -f "$HOME/.bashrc" ] && . "$HOME/.bashrc"`
	var profile string
	var prompt string
	switch shellName {
	case "bash":
		if runtime.GOOS != "darwin" {
			return nil
		}
		profile = filepath.Join(cfg.home, ".bash_profile")
		prompt = "On macOS, bash login shells read ~/.bash_profile. Source ~/.bashrc from it?"
	case "zsh":
		profile = filepath.Join(cfg.home, ".zshrc")
		prompt = "Your shell appears to be zsh. Source ~/.bashrc from ~/.zshrc too?"
	default:
		return nil
	}
	content, _ := os.ReadFile(profile)
	if strings.Contains(string(content), ".bashrc") {
		return nil
	}
	if !promptYesNo(prompt, true) {
		return nil
	}
	f, err := os.OpenFile(profile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "\n%s\n", sourceLine); err != nil {
		return err
	}
	fmt.Printf("Updated %s to source ~/.bashrc.\n", profile)
	return nil
}

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

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			os.Remove(target)
			return os.Symlink(link, target)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func extractTarGZStripFirstComponent(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		parts := strings.Split(header.Name, "/")
		if len(parts) <= 1 {
			continue
		}
		rel := filepath.Join(parts[1:]...)
		if rel == "." || rel == "" {
			continue
		}
		target := filepath.Join(destDir, rel)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("archive path escapes destination: %s", header.Name)
		}
		mode := os.FileMode(header.Mode).Perm()
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		}
	}
}

func doctor(cfg config) error {
	if err := assertSupportedOS(); err != nil {
		return err
	}
	fmt.Printf("OS: %s %s\n", runtime.GOOS, runtime.GOARCH)
	if haveCmd("curl") || haveCmd("wget") {
		fmt.Println("ok      curl or wget")
	} else {
		fmt.Println("missing curl or wget")
	}
	if commandExistsWithNVM(cfg, "node") {
		out, _ := captureShellWithNVM(cfg, "node --version")
		fmt.Printf("node: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("node: missing")
	}
	if commandExistsWithNVM(cfg, "npm") {
		version, _ := captureShellWithNVM(cfg, "npm --version")
		prefix, _ := npmGlobalPrefix(cfg)
		fmt.Printf("npm: %s\n", strings.TrimSpace(version))
		fmt.Printf("npm prefix: %s\n", prefix)
	} else {
		fmt.Println("npm: missing")
	}
	if commandExistsWithNVM(cfg, "claude") {
		out, _ := captureShellWithNVM(cfg, "claude --version")
		fmt.Printf("claude: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("claude: missing")
	}
	if commandExistsWithNVM(cfg, "codex") {
		out, _ := captureShellWithNVM(cfg, "codex --version")
		fmt.Printf("codex: %s\n", strings.TrimSpace(out))
	} else {
		fmt.Println("codex: missing")
	}
	fmt.Printf("claude skills dir: %s\n", cfg.claudeSkillsDir)
	fmt.Printf("codex skills dir: %s\n", cfg.codexSkillsDir)
	fmt.Printf("api key rc file: %s\n", cfg.rcFile)
	return nil
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

func commandExistsWithNVM(cfg config, name string) bool {
	_, err := captureShellWithNVM(cfg, "command -v "+shellQuote(name))
	return err == nil
}

func captureShellWithNVM(cfg config, command string) (string, error) {
	script := nvmShellPrefix(cfg) + command
	cmd := exec.Command("bash", "-lc", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

func runShellWithNVM(cfg config, command string) error {
	return runCommand("bash", "-lc", nvmShellPrefix(cfg)+command)
}

func nvmShellPrefix(cfg config) string {
	return "export NVM_DIR=" + shellQuote(cfg.nvmDir) + "; [ -s \"$NVM_DIR/nvm.sh\" ] && . \"$NVM_DIR/nvm.sh\"; "
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func runCommandWithIO(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func haveCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

func stdinIsTTY() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
