package main

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestATerminalThatWontOpenIsLoggedWithoutTheItems(t *testing.T) {
	const detachedProcess = 0x00000008 // no console, so CONIN$ can't open
	noConsole := func(cmd *exec.Cmd) {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
	}
	failed(t, fzt(t, "private-item\n", noConsole, "--border"), "couldn't open the terminal: ", "private-item")
}
