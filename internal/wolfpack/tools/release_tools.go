package tools

// release_tools.go installs non-npm developer tools from official releases.

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
	"github.com/ayushag-nv/wolfpack/internal/wolfpack/system"
)

const uvInstallerURL = "https://astral.sh/uv/%s/install.sh"

// ReleaseTool describes a non-npm tool with its version source.
type ReleaseTool struct {
	Target         string
	Label          string
	BinaryName     string
	VersionCommand string
	fetchVersions  func(int) (string, []string, error)
}

// ReleaseTools is the list of developer tools Wolfpack installs outside npm.
var ReleaseTools = []ReleaseTool{
	{Target: "uv", Label: "uv", BinaryName: "uv", VersionCommand: "uv --version", fetchVersions: uvVersions},
	{Target: "ruff", Label: "Ruff", BinaryName: "ruff", VersionCommand: "ruff --version", fetchVersions: ruffVersions},
	{Target: "gh", Label: "GitHub CLI", BinaryName: "gh", VersionCommand: "gh --version", fetchVersions: ghVersions},
	{Target: "glab", Label: "GitLab CLI", BinaryName: "glab", VersionCommand: "glab --version", fetchVersions: glabVersions},
}

// ReleaseToolByTarget returns metadata for a normalized non-npm target.
func ReleaseToolByTarget(target string) (ReleaseTool, bool) {
	for _, tool := range ReleaseTools {
		if tool.Target == target {
			return tool, true
		}
	}
	return ReleaseTool{}, false
}

// InstallUV installs uv with Astral's standalone installer.
func InstallUV(cfg config.Config) error {
	latest, versions, err := uvVersions(cfg.VersionLimit)
	if err != nil {
		return err
	}
	version, err := chooseConcreteVersion("uv", latest, versions)
	if err != nil {
		return err
	}
	url := fmt.Sprintf(uvInstallerURL, version)
	return installShellInstaller(cfg, "uv", "uv", url, map[string]string{
		"UV_INSTALL_DIR":    cfg.BinDir,
		"UV_NO_MODIFY_PATH": "1",
	})
}

// InstallRuff installs Ruff as an isolated uv-managed tool.
func InstallRuff(cfg config.Config) error {
	if !system.CommandExistsWithUserBin(cfg, "uv") {
		fmt.Println("Ruff uses uv for isolated tool installs; installing uv first.")
		if err := InstallUV(cfg); err != nil {
			return err
		}
	}
	latest, versions, err := ruffVersions(cfg.VersionLimit)
	if err != nil {
		return err
	}
	version, err := chooseConcreteVersion("Ruff", latest, versions)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.BinDir, 0o755); err != nil {
		return err
	}
	spec := "ruff@" + version
	fmt.Printf("Installing Ruff with uv tool package %s.\n", spec)
	if err := system.RunShellWithUserBin(cfg, "UV_TOOL_BIN_DIR="+system.ShellQuote(cfg.BinDir)+" uv tool install "+system.ShellQuote(spec)); err != nil {
		return err
	}
	return finishUserBinInstall(cfg, "Ruff", "ruff", "ruff --version")
}

// InstallGH installs GitHub CLI from official release binaries.
func InstallGH(cfg config.Config) error {
	latest, versions, err := ghVersions(cfg.VersionLimit)
	if err != nil {
		return err
	}
	version, err := chooseConcreteVersion("GitHub CLI", latest, versions)
	if err != nil {
		return err
	}
	url, member, err := ghArchive(version)
	if err != nil {
		return err
	}
	return installReleaseArchiveTool(cfg, "GitHub CLI", "gh", "gh --version", url, member)
}

// InstallGLab installs GitLab CLI from official release binaries.
func InstallGLab(cfg config.Config) error {
	latest, versions, err := glabVersions(cfg.VersionLimit)
	if err != nil {
		return err
	}
	version, err := chooseConcreteVersion("GitLab CLI", latest, versions)
	if err != nil {
		return err
	}
	url, member, err := glabArchive(version)
	if err != nil {
		return err
	}
	return installReleaseArchiveTool(cfg, "GitLab CLI", "glab", "glab --version", url, member)
}

// ListReleaseVersions prints recent versions for a non-npm tool.
func ListReleaseVersions(cfg config.Config, target string) error {
	tool, ok := ReleaseToolByTarget(target)
	if !ok {
		return fmt.Errorf("unknown release target: %s", target)
	}
	latest, versions, err := tool.fetchVersions(cfg.VersionLimit)
	if err != nil {
		return err
	}
	fmt.Printf("%s versions:\n", tool.Label)
	fmt.Printf("latest: %s\n", latest)
	for _, version := range versions {
		fmt.Printf("  %s\n", version)
	}
	return nil
}

// installShellInstaller downloads and runs a vendor shell installer safely.
func installShellInstaller(cfg config.Config, label, binaryName, url string, env map[string]string) error {
	if err := system.AssertSupportedOS(); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.BinDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "wolfpack-installer-*.sh")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	fmt.Printf("Downloading %s installer from %s.\n", label, url)
	if err := system.DownloadFile(url, tmp.Name()); err != nil {
		return err
	}
	var assignments []string
	for key, value := range env {
		assignments = append(assignments, key+"="+system.ShellQuote(value))
	}
	sort.Strings(assignments)
	command := strings.Join(append(assignments, "sh", system.ShellQuote(tmp.Name())), " ")
	if err := system.RunShellWithUserBin(cfg, command); err != nil {
		return err
	}
	return finishUserBinInstall(cfg, label, binaryName, binaryName+" --version")
}

// installReleaseArchiveTool installs one binary from a tar.gz or zip archive.
func installReleaseArchiveTool(cfg config.Config, label, binaryName, versionCommand, url, member string) error {
	if err := system.AssertSupportedOS(); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.BinDir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "wolfpack-archive-*")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	fmt.Printf("Downloading %s release from %s.\n", label, url)
	if err := system.DownloadFile(url, tmp.Name()); err != nil {
		return err
	}
	dest := filepath.Join(cfg.BinDir, binaryName)
	if strings.HasSuffix(url, ".zip") {
		err = system.ExtractFileFromZip(tmp.Name(), member, dest)
	} else {
		err = system.ExtractFileFromTarGZ(tmp.Name(), member, dest)
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return err
	}
	return finishUserBinInstall(cfg, label, binaryName, versionCommand)
}

// finishUserBinInstall updates PATH setup and prints the installed version.
func finishUserBinInstall(cfg config.Config, label, binaryName, versionCommand string) error {
	if err := system.EnsurePathEntryInRC(cfg, cfg.BinDir); err != nil {
		return err
	}
	if system.CommandExistsWithUserBin(cfg, binaryName) {
		fmt.Printf("%s installed:\n", label)
		_ = system.RunShellWithUserBin(cfg, versionCommand)
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

// chooseConcreteVersion prompts for an exact release version and returns a version number.
func chooseConcreteVersion(label, latest string, versions []string) (string, error) {
	if latest == "" {
		return "", fmt.Errorf("could not determine latest version for %s", label)
	}
	if !system.StdinIsTTY() {
		return latest, nil
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
		choice = strings.TrimPrefix(strings.TrimSpace(choice), "v")
		if choice == "" || choice == "0" {
			return latest, nil
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

// uvVersions returns recent uv versions from the GitHub releases feed.
func uvVersions(limit int) (string, []string, error) {
	return githubAtomVersions("https://github.com/astral-sh/uv/releases.atom", limit)
}

// ghVersions returns recent GitHub CLI versions from the GitHub releases feed.
func ghVersions(limit int) (string, []string, error) {
	return githubAtomVersions("https://github.com/cli/cli/releases.atom", limit)
}

// ruffVersions returns recent Ruff versions from PyPI.
func ruffVersions(limit int) (string, []string, error) {
	return pypiVersionsForPackage("ruff", limit)
}

// glabVersions returns recent GitLab CLI versions from the GitLab releases API.
func glabVersions(limit int) (string, []string, error) {
	url := fmt.Sprintf("https://gitlab.com/api/v4/projects/gitlab-org%%2Fcli/releases?per_page=%d", limit)
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("GitLab returned HTTP %d", resp.StatusCode)
	}
	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", nil, err
	}
	var versions []string
	for _, release := range releases {
		version := strings.TrimPrefix(release.TagName, "v")
		if parseSemver(version) != nil {
			versions = append(versions, version)
		}
	}
	sortVersionsNewestFirst(versions)
	if len(versions) > limit {
		versions = versions[:limit]
	}
	if len(versions) == 0 {
		return "", nil, fmt.Errorf("no GitLab CLI versions found")
	}
	return versions[0], versions, nil
}

// pypiVersionsForPackage returns recent stable versions from PyPI metadata.
func pypiVersionsForPackage(packageName string, limit int) (string, []string, error) {
	url := "https://pypi.org/pypi/" + packageName + "/json"
	resp, err := http.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("PyPI returned HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		Releases map[string]json.RawMessage `json:"releases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", nil, err
	}
	var versions []string
	for version := range payload.Releases {
		if stablePattern.MatchString(version) {
			versions = append(versions, version)
		}
	}
	sortVersionsNewestFirst(versions)
	if len(versions) > limit {
		versions = versions[:limit]
	}
	return payload.Info.Version, versions, nil
}

// githubAtomVersions returns recent semantic release tags from a GitHub Atom feed.
func githubAtomVersions(feedURL string, limit int) (string, []string, error) {
	resp, err := http.Get(feedURL)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}
	var feed struct {
		Entries []struct {
			Link struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return "", nil, err
	}
	var versions []string
	seen := make(map[string]bool)
	for _, entry := range feed.Entries {
		index := strings.LastIndex(entry.Link.Href, "/tag/")
		if index < 0 {
			continue
		}
		version := strings.TrimPrefix(entry.Link.Href[index+len("/tag/"):], "v")
		if parseSemver(version) == nil || seen[version] {
			continue
		}
		seen[version] = true
		versions = append(versions, version)
	}
	sortVersionsNewestFirst(versions)
	if len(versions) > limit {
		versions = versions[:limit]
	}
	if len(versions) == 0 {
		return "", nil, fmt.Errorf("no versions found in %s", feedURL)
	}
	return versions[0], versions, nil
}

// sortVersionsNewestFirst sorts semantic versions descending in place.
func sortVersionsNewestFirst(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return compareSemver(versions[i], versions[j]) > 0
	})
}

// ghArchive returns the official release archive URL and executable member path.
func ghArchive(version string) (string, string, error) {
	arch, err := releaseArch(runtime.GOARCH)
	if err != nil {
		return "", "", err
	}
	switch runtime.GOOS {
	case "linux":
		return fmt.Sprintf("https://github.com/cli/cli/releases/download/v%s/gh_%s_linux_%s.tar.gz", version, version, arch), "bin/gh", nil
	case "darwin":
		return fmt.Sprintf("https://github.com/cli/cli/releases/download/v%s/gh_%s_macOS_%s.zip", version, version, arch), "bin/gh", nil
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// glabArchive returns the official release archive URL and executable member path.
func glabArchive(version string) (string, string, error) {
	arch, err := releaseArch(runtime.GOARCH)
	if err != nil {
		return "", "", err
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		return fmt.Sprintf("https://gitlab.com/gitlab-org/cli/-/releases/v%s/downloads/glab_%s_%s_%s.tar.gz", version, version, runtime.GOOS, arch), "bin/glab", nil
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// releaseArch maps Go architecture names to release asset names.
func releaseArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}
}
