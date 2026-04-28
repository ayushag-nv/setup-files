package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func runWolfpack(t *testing.T, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command(goBinary(t), append([]string{"run", "../cmd/wolfpack"}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wolfpack %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func goBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("go binary not found at %s: %v", path, err)
	}
	return path
}

func TestHelpMentionsOpenCode(t *testing.T) {
	out := runWolfpack(t, nil, "help")
	for _, want := range []string{
		"wolfpack install [all|claude|codex|code|opencode|skills]",
		"wolfpack versions [claude|codex|code|opencode]",
		"OpenCode",
		"OPENCODE_SKILLS_DIR",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q\n%s", want, out)
		}
	}
}

func TestVersionOutput(t *testing.T) {
	out := runWolfpack(t, nil, "--version")
	if !regexp.MustCompile(`^wolfpack \d+\.\d+\.\d+\n$`).MatchString(out) {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestSkillsInstallIncludesOpenCodeDestination(t *testing.T) {
	tmp := t.TempDir()
	source := filepath.Join(tmp, "source")
	sourceSkills := filepath.Join(source, "skills")

	writeSkill(t, sourceSkills, "example-skill")
	writeSkill(t, sourceSkills, "ultimate-auto")

	claudeDir := filepath.Join(tmp, "claude")
	codexDir := filepath.Join(tmp, "codex")
	opencodeDir := filepath.Join(tmp, "opencode")

	out := runWolfpack(t, []string{
		"WOLFPACK_SKILLS_SOURCE=" + source,
		"CLAUDE_SKILLS_DIR=" + claudeDir,
		"CODEX_SKILLS_DIR=" + codexDir,
		"OPENCODE_SKILLS_DIR=" + opencodeDir,
	}, "skills", "install")

	if !strings.Contains(out, "Installed 1 skills for OpenCode") {
		t.Fatalf("skills install output missing OpenCode install line\n%s", out)
	}

	for _, dir := range []string{claudeDir, codexDir, opencodeDir} {
		if _, err := os.Stat(filepath.Join(dir, "example-skill", "SKILL.md")); err != nil {
			t.Fatalf("expected example skill in %s: %v", dir, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "ultimate-auto", "SKILL.md")); !os.IsNotExist(err) {
			t.Fatalf("excluded ultimate-auto was installed in %s", dir)
		}
	}
}

func writeSkill(t *testing.T, skillsDir, name string) {
	t.Helper()
	dir := filepath.Join(skillsDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Test skill\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
