package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type npmMetadata struct {
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]json.RawMessage `json:"versions"`
}

type semver struct {
	major      int
	minor      int
	patch      int
	prerelease string
}

var (
	semverPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)
	stablePattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
)

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
		if stablePattern.MatchString(version) {
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
	return &semver{major: major, minor: minor, patch: patch, prerelease: match[4]}
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

	reader := newInputReader()
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
