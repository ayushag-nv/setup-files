package system

// process.go wraps shell command execution, especially commands needing nvm.

import (
	"io"
	"os"
	"os/exec"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
)

// CommandExistsWithNVM checks PATH after loading nvm shell initialization.
func CommandExistsWithNVM(cfg config.Config, name string) bool {
	_, err := CaptureShellWithNVM(cfg, "command -v "+ShellQuote(name))
	return err == nil
}

// CommandExistsWithUserBin checks PATH with Wolfpack's bin directory prepended.
func CommandExistsWithUserBin(cfg config.Config, name string) bool {
	_, err := CaptureShellWithUserBin(cfg, "command -v "+ShellQuote(name))
	return err == nil
}

// CaptureShellWithNVM runs a bash command with nvm loaded and returns output.
func CaptureShellWithNVM(cfg config.Config, command string) (string, error) {
	script := nvmShellPrefix(cfg) + command
	cmd := exec.Command("bash", "-lc", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// CaptureShellWithUserBin runs a bash command with Wolfpack's bin dir on PATH.
func CaptureShellWithUserBin(cfg config.Config, command string) (string, error) {
	script := userBinShellPrefix(cfg) + command
	cmd := exec.Command("bash", "-lc", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// RunShellWithNVM streams a bash command after loading nvm.
func RunShellWithNVM(cfg config.Config, command string) error {
	return RunCommand("bash", "-lc", nvmShellPrefix(cfg)+command)
}

// RunShellWithUserBin streams a bash command with Wolfpack's bin dir on PATH.
func RunShellWithUserBin(cfg config.Config, command string) error {
	return RunCommand("bash", "-lc", userBinShellPrefix(cfg)+command)
}

// nvmShellPrefix builds the shell snippet that loads nvm when available.
func nvmShellPrefix(cfg config.Config) string {
	return "export NVM_DIR=" + ShellQuote(cfg.NVMDir) + "; [ -s \"$NVM_DIR/nvm.sh\" ] && . \"$NVM_DIR/nvm.sh\"; "
}

// userBinShellPrefix builds the shell snippet that exposes Wolfpack-installed tools.
func userBinShellPrefix(cfg config.Config) string {
	return "export PATH=" + ShellQuote(cfg.BinDir) + ":$PATH; "
}

// RunCommand runs a subprocess using the current terminal for all IO.
func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunCommandQuiet runs a subprocess without attaching terminal output.
func RunCommandQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// RunCommandWithIO runs a subprocess with caller-provided IO streams.
func RunCommandWithIO(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// HaveCmd checks whether a command exists in the current process PATH.
func HaveCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
