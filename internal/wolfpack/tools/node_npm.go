package tools

// Package tools installs developer CLIs and discovers available versions.

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

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
)

const (
	claudePackage   = "@anthropic-ai/claude-code"
	codexPackage    = "@openai/codex"
	opencodePackage = "opencode-ai"
	minNodeMajor    = 18
)

// NPMTool describes one global npm package and the command it should expose.
type NPMTool struct {
	Target      string
	Label       string
	PackageName string
	BinaryName  string
}

// NPMTools is the list of AI coding CLIs Wolfpack installs through npm.
var NPMTools = []NPMTool{
	{Target: "claude", Label: "Claude Code", PackageName: claudePackage, BinaryName: "claude"},
	{Target: "codex", Label: "OpenAI Codex CLI", PackageName: codexPackage, BinaryName: "codex"},
	{Target: "opencode", Label: "OpenCode CLI", PackageName: opencodePackage, BinaryName: "opencode"},
}

// NPMToolByTarget returns metadata for a normalized npm-backed install target.
func NPMToolByTarget(target string) (NPMTool, bool) {
	for _, tool := range NPMTools {
		if tool.Target == target {
			return tool, true
		}
	}
	return NPMTool{}, false
}

// InstallNPMTarget installs a normalized npm-backed Wolfpack target.
func InstallNPMTarget(cfg config.Config, target string) error {
	tool, ok := NPMToolByTarget(target)
	if !ok {
		return fmt.Errorf("unknown npm target: %s", target)
	}
	return InstallNPMTool(cfg, tool.Label, tool.PackageName, tool.BinaryName)
}

// npmMetadata is the subset of npm registry metadata Wolfpack needs.
type npmMetadata struct {
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]json.RawMessage `json:"versions"`
}

// semver stores parsed semantic-version components for sorting.
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

// InstallNPMTool installs one selected npm package globally and verifies its binary.
func InstallNPMTool(cfg config.Config, label, packageName, binaryName string) error {
	if err := system.AssertSupportedOS(); err != nil {
		return err
	}
	if err := EnsureNode(cfg); err != nil {
		return err
	}
	if err := EnsureNPMCLIPath(cfg); err != nil {
		return err
	}

	version, err := choosePackageVersion(cfg, label, packageName)
	if err != nil {
		return err
	}
	spec := packageName + "@" + version
	fmt.Printf("Installing %s with npm package %s.\n", label, spec)
	if err := system.RunShellWithNVM(cfg, "npm install -g "+system.ShellQuote(spec)); err != nil {
		return err
	}
	if err := EnsureNPMCLIPath(cfg); err != nil {
		return err
	}

	if system.CommandExistsWithNVM(cfg, binaryName) {
		fmt.Printf("%s installed:\n", label)
		_ = system.RunShellWithNVM(cfg, binaryName+" --version")
	} else {
		system.Warn("%s install completed, but %q is not on PATH in this shell", label, binaryName)
	}

	if err := system.InstallShellWrapper(cfg); err != nil {
		return err
	}
	if err := system.MaybeSourceBashrcFromShellProfile(cfg); err != nil {
		return err
	}
	fmt.Printf("For this terminal, run once: source %s\n", cfg.RCFile)
	return nil
}

// EnsureNode ensures usable Node.js and npm are available without sudo npm.
func EnsureNode(cfg config.Config) error {
	if system.CommandExistsWithNVM(cfg, "node") && system.CommandExistsWithNVM(cfg, "npm") {
		major, err := nodeMajor(cfg)
		if err == nil && major >= minNodeMajor && npmGlobalPrefixWritable(cfg) {
			return nil
		}
		if err == nil && major < minNodeMajor {
			system.Warn("Node.js %d+ is required; current major is %d", minNodeMajor, major)
		} else if !npmGlobalPrefixWritable(cfg) {
			system.Warn("npm global prefix is not writable; installing user-local Node.js through nvm")
		}
	} else {
		fmt.Println("Node.js and npm are required; installing them through nvm.")
	}

	if err := installNVMAndNode(cfg); err != nil {
		return err
	}
	if !system.CommandExistsWithNVM(cfg, "node") {
		return errors.New("node was not found after installation")
	}
	if !system.CommandExistsWithNVM(cfg, "npm") {
		return errors.New("npm was not found after installation")
	}
	return nil
}

// installNVMAndNode installs nvm if needed, then installs latest Node LTS.
func installNVMAndNode(cfg config.Config) error {
	if !system.CommandExistsWithNVM(cfg, "nvm") {
		version := os.Getenv("NVM_VERSION")
		if version == "" {
			latest, err := latestNVMVersion()
			if err == nil {
				version = latest
			}
		}
		if version == "" {
			version = "v0.40.3"
			system.Warn("could not detect latest nvm release; falling back to %s", version)
		}

		fmt.Printf("Installing nvm %s.\n", version)
		tmp, err := os.CreateTemp("", "wolfpack-nvm-*.sh")
		if err != nil {
			return err
		}
		tmp.Close()
		defer os.Remove(tmp.Name())

		url := fmt.Sprintf("https://raw.githubusercontent.com/nvm-sh/nvm/%s/install.sh", version)
		if err := system.DownloadFile(url, tmp.Name()); err != nil {
			return err
		}
		if err := system.RunCommand("bash", tmp.Name()); err != nil {
			return err
		}
	}

	fmt.Println("Installing latest Node.js LTS through nvm.")
	if err := system.RunShellWithNVM(cfg, "nvm install --lts && nvm alias default 'lts/*' >/dev/null && nvm use --lts >/dev/null"); err != nil {
		return err
	}
	return system.EnsureNVMShellInit(cfg)
}

// latestNVMVersion fetches the latest nvm release tag from GitHub.
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

// nodeMajor returns the current Node.js major version after loading nvm.
func nodeMajor(cfg config.Config) (int, error) {
	out, err := system.CaptureShellWithNVM(cfg, "node -p \"process.versions.node.split('.')[0]\"")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// NPMGlobalPrefix returns the npm global install prefix.
func NPMGlobalPrefix(cfg config.Config) (string, error) {
	out, err := system.CaptureShellWithNVM(cfg, "npm config get prefix")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// npmGlobalPrefixWritable reports whether global npm installs can write there.
func npmGlobalPrefixWritable(cfg config.Config) bool {
	prefix, err := NPMGlobalPrefix(cfg)
	if err != nil || prefix == "" {
		return false
	}
	info, err := os.Stat(prefix)
	if err == nil && info.IsDir() {
		return system.CanWriteDir(prefix)
	}
	parent := filepath.Dir(prefix)
	if parent == "." || parent == prefix {
		return false
	}
	return system.CanWriteDir(parent)
}

// EnsureNPMCLIPath makes globally installed npm binaries discoverable later.
func EnsureNPMCLIPath(cfg config.Config) error {
	if !system.CommandExistsWithNVM(cfg, "npm") {
		return nil
	}
	prefix, err := NPMGlobalPrefix(cfg)
	if err != nil || prefix == "" {
		return err
	}
	binDir := filepath.Join(prefix, "bin")
	if strings.HasPrefix(binDir, cfg.NVMDir+string(os.PathSeparator)) {
		return system.EnsureNVMShellInit(cfg)
	}
	return system.EnsurePathEntryInRC(cfg, binDir)
}

// NPMVersionsForPackage fetches latest and recent semver versions from npm.
func NPMVersionsForPackage(packageName string, limit int) (string, []string, error) {
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

// parseSemver parses a version string accepted by Wolfpack's version chooser.
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

// compareSemver orders semantic versions from oldest to newest.
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

// choosePackageVersion prompts for an npm package version or defaults to latest.
func choosePackageVersion(cfg config.Config, label, packageName string) (string, error) {
	fmt.Fprintf(os.Stderr, "Fetching available %s versions from npm registry.\n", label)
	latest, versions, err := NPMVersionsForPackage(packageName, cfg.VersionLimit)
	if err != nil {
		return "", err
	}
	if latest == "" {
		return "", fmt.Errorf("could not determine latest version for %s", packageName)
	}
	if !system.StdinIsTTY() {
		return "latest", nil
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s versions:\n", label)
	fmt.Fprintf(os.Stderr, "  0) latest (%s)\n", latest)
	for i, version := range versions {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, version)
	}
	fmt.Fprintf(os.Stderr, "  or enter an exact version, for example %s\n", latest)

	reader := system.NewInputReader()
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
		system.Warn("enter a number between 0 and %d, or an exact version", len(versions))
	}
}

// ListVersions prints recent versions for one tool target or all tool targets.
func ListVersions(cfg config.Config, target string) error {
	switch target {
	case "all":
		for i, tool := range NPMTools {
			if i > 0 {
				fmt.Println()
			}
			if err := ListVersions(cfg, tool.Target); err != nil {
				return err
			}
		}
		for _, tool := range ReleaseTools {
			fmt.Println()
			if err := ListReleaseVersions(cfg, tool.Target); err != nil {
				return err
			}
		}
		return nil
	}

	tool, ok := NPMToolByTarget(target)
	if ok {
		if err := EnsureNode(cfg); err != nil {
			return err
		}
		latest, versions, err := NPMVersionsForPackage(tool.PackageName, cfg.VersionLimit)
		if err != nil {
			return err
		}
		fmt.Printf("%s npm versions:\n", tool.Label)
		fmt.Printf("latest: %s\n", latest)
		for _, version := range versions {
			fmt.Printf("  %s\n", version)
		}
		return nil
	}
	if _, ok := ReleaseToolByTarget(target); ok {
		return ListReleaseVersions(cfg, target)
	}
	return fmt.Errorf("unknown target: %s", target)
}
