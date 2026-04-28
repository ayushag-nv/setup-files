package main

import (
	"io"
	"os"
	"os/exec"
)

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
