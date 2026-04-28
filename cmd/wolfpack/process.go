package main

// process.go wraps shell command execution, especially commands needing nvm.

import (
	"io"
	"os"
	"os/exec"
)

// commandExistsWithNVM checks PATH after loading nvm shell initialization.
func commandExistsWithNVM(cfg config, name string) bool {
	_, err := captureShellWithNVM(cfg, "command -v "+shellQuote(name))
	return err == nil
}

// captureShellWithNVM runs a bash command with nvm loaded and returns output.
func captureShellWithNVM(cfg config, command string) (string, error) {
	script := nvmShellPrefix(cfg) + command
	cmd := exec.Command("bash", "-lc", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// runShellWithNVM streams a bash command after loading nvm.
func runShellWithNVM(cfg config, command string) error {
	return runCommand("bash", "-lc", nvmShellPrefix(cfg)+command)
}

// nvmShellPrefix builds the shell snippet that loads nvm when available.
func nvmShellPrefix(cfg config) string {
	return "export NVM_DIR=" + shellQuote(cfg.nvmDir) + "; [ -s \"$NVM_DIR/nvm.sh\" ] && . \"$NVM_DIR/nvm.sh\"; "
}

// runCommand runs a subprocess using the current terminal for all IO.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runCommandQuiet runs a subprocess without attaching terminal output.
func runCommandQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// runCommandWithIO runs a subprocess with caller-provided IO streams.
func runCommandWithIO(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// haveCmd checks whether a command exists in the current process PATH.
func haveCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
